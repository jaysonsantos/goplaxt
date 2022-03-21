package api

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/xanderstrike/goplaxt/lib/store"
	"github.com/xanderstrike/goplaxt/lib/trakt"
	"github.com/xanderstrike/goplaxt/tracing"
	"github.com/xanderstrike/plexhooks"

	log "github.com/sirupsen/logrus"
)

const maxMemory = 6 * 1024 * 1024

var storage store.Store

func SetStore(s store.Store) {
	storage = s
}

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.Tracer.Start(r.Context(), "api")
	defer span.End()
	logger := log.WithContext(ctx)
	args := r.URL.Query()
	if log.GetLevel() == log.DebugLevel {
		fields := make(map[string]interface{}, len(args))
		for k, v := range args {
			fields[k] = v
		}
		logger = logger.WithFields(fields)
	} else {
		logger = logger.WithField("request", r.URL.Path)
	}
	id := args["id"][0]
	logger.Printf("Webhook call for %s", id)

	user, err := storage.GetUser(id)
	if err != nil {
		logger.Errorf("error getting user: %#v", err)
		http.Error(w, "Failed to find a valid user", http.StatusInternalServerError)
		return
	}

	if user == nil {
		log.Println("User not found.")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode("user not found")
		return
	}

	logger = logger.WithField("user", user.ID)

	tokenAge := time.Since(user.Updated).Hours()
	if tokenAge > 1440 { // tokens expire after 3 months, so we refresh after 2
		logger.Println("User access token outdated, refreshing...")
		result, err := trakt.AuthRequest(SelfRoot(r), user.Username, "", user.RefreshToken, "refresh_token")
		if err != nil {
			logger.Println(fmt.Errorf("refresh failed, skipping and deleting user %w", err))
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("fail")
			storage.DeleteUser(user.ID)
			return
		}

		user.UpdateUser(result["access_token"].(string), result["refresh_token"].(string))
		logger.Println("Refreshed, continuing")

	}

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		logger.Errorf("error reading body: %#v", err)
		http.Error(w, "Failed to read webhook body", http.StatusInternalServerError)
		return
	}

	multipart.NewReader(r.Body, r.Header.Get("Content-Type"))

	re, err := plexhooks.ParseWebhook([]byte(r.PostFormValue("payload")))
	if err != nil {
		logger.Errorf("failed to process webhook: %#v\n%s", err, r.PostFormValue("payload"))
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	if strings.ToLower(re.Account.Title) == user.Username {
		// FIXME - make everything take the pointer
		// Don't let plex waiting
		go trakt.Handle(re, *user, logger)
	} else {
		logger.Errorf("Plex username %s does not equal %s, skipping", strings.ToLower(re.Account.Title), user.Username)
	}

	json.NewEncoder(w).Encode("success")
}
