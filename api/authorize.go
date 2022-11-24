package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/xanderstrike/goplaxt/lib/store"
	"github.com/xanderstrike/goplaxt/lib/trakt"

	log "github.com/sirupsen/logrus"
)

type AuthorizePage struct {
	SelfRoot   string
	Authorized bool
	URL        string
	ClientID   string
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	args := r.URL.Query()
	username := strings.ToLower(args["username"][0])
	log.Print(fmt.Sprintf("Handling auth request for %s", username))
	code := args["code"][0]
	result, err := trakt.AuthRequest(ctx, SelfRoot(r), username, code, "", "authorization_code")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := store.NewUser(ctx, username, result["access_token"].(string), result["refresh_token"].(string), storage)
	if err != nil {
		log.Errorf("error saving user: %#v", err)
		http.Error(w, "Failed to write user credentials", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("%s/api?id=%s", SelfRoot(r), user.ID)

	log.Print(fmt.Sprintf("Authorized as %s", user.ID))

	tmpl := template.Must(template.ParseFiles("static/index.html"))
	data := AuthorizePage{
		SelfRoot:   SelfRoot(r),
		Authorized: true,
		URL:        url,
		ClientID:   os.Getenv("TRAKT_ID"),
	}
	tmpl.Execute(w, data)
}
