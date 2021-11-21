package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/etherlabsio/healthcheck"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/xanderstrike/goplaxt/lib/store"
	"github.com/xanderstrike/goplaxt/lib/trakt"
	"github.com/xanderstrike/plexhooks"
)

const maxMemory = 6 * 1024 * 1024

var storage store.Store

type AuthorizePage struct {
	SelfRoot   string
	Authorized bool
	URL        string
	ClientID   string
}

func SelfRoot(r *http.Request) string {
	u, _ := url.Parse("")
	u.Host = r.Host
	u.Scheme = r.URL.Scheme
	u.Path = ""
	if u.Scheme == "" {
		u.Scheme = "http"

		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "https" {
			u.Scheme = "https"
		}
	}
	return u.String()
}

func authorize(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	username := strings.ToLower(args["username"][0])
	log.Print(fmt.Sprintf("Handling auth request for %s", username))
	code := args["code"][0]
	result, err := trakt.AuthRequest(SelfRoot(r), username, code, "", "authorization_code")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := store.NewUser(username, result["access_token"].(string), result["refresh_token"].(string), storage)
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

func api(w http.ResponseWriter, r *http.Request) {
	var logger *log.Entry
	args := r.URL.Query()
	if log.GetLevel() == log.DebugLevel {
		fields := make(map[string]interface{}, len(args))
		for k, v := range args {
			fields[k] = v
		}
		logger = log.WithFields(fields)
	} else {
		logger = log.WithField("request", r.URL.Path)
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
		logger.Errorf("failed to process webhook: %#v", err)
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

func allowedHostsHandler(allowedHostnames string) func(http.Handler) http.Handler {
	allowedHosts := strings.Split(regexp.MustCompile(`https://|http://|\s+`).ReplaceAllString(strings.ToLower(allowedHostnames), ""), ",")
	log.Println("Allowed Hostnames:", allowedHosts)
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.EscapedPath() == "/healthcheck" {
				h.ServeHTTP(w, r)
				return
			}
			isAllowedHost := false
			lcHost := strings.ToLower(r.Host)
			for _, value := range allowedHosts {
				if lcHost == value {
					isAllowedHost = true
					break
				}
			}
			if !isAllowedHost {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, "Oh no!")
				return
			}
			h.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func healthcheckHandler() http.Handler {
	return healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker("storage", healthcheck.CheckerFunc(func(ctx context.Context) error {
			return storage.Ping(ctx)
		})),
	)
}

func main() {
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		parsedLogLevel, err := log.ParseLevel(logLevel)
		if err != nil {
			log.WithField("logLevel", logLevel).Error("failed to parse log level")
			parsedLogLevel = log.InfoLevel
		}
		log.SetLevel(parsedLogLevel)
	}
	log.WithField("logLevel", log.GetLevel().String()).Print("Started!")
	if os.Getenv("POSTGRESQL_URL") != "" {
		storage = store.NewPostgresqlStore(store.NewPostgresqlClient(os.Getenv("POSTGRESQL_URL")))
		log.Println("Using postgresql storage:", os.Getenv("POSTGRESQL_URL"))
	} else if os.Getenv("REDIS_URI") != "" {
		storage = store.NewRedisStore(store.NewRedisClient(os.Getenv("REDIS_URI"), os.Getenv("REDIS_PASSWORD")))
		log.Println("Using redis storage:", os.Getenv("REDIS_URI"))
	} else {
		storage = store.NewDiskStore()
		log.Println("Using disk storage:")
	}

	router := mux.NewRouter()
	// Assumption: Behind a proper web server (nginx/traefik, etc) that removes/replaces trusted headers
	router.Use(handlers.ProxyHeaders)
	// which hostnames we are allowing
	// REDIRECT_URI = old legacy list
	// ALLOWED_HOSTNAMES = new accurate config variable
	// No env = all hostnames
	if os.Getenv("REDIRECT_URI") != "" {
		router.Use(allowedHostsHandler(os.Getenv("REDIRECT_URI")))
	} else if os.Getenv("ALLOWED_HOSTNAMES") != "" {
		router.Use(allowedHostsHandler(os.Getenv("ALLOWED_HOSTNAMES")))
	}
	router.HandleFunc("/authorize", authorize).Methods("GET")
	router.HandleFunc("/api", api).Methods("POST")
	router.Handle("/healthcheck", healthcheckHandler()).Methods("GET")
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("static/index.html"))
		data := AuthorizePage{
			SelfRoot:   SelfRoot(r),
			Authorized: false,
			URL:        "https://plaxt.astandke.com/api?id=generate-your-own-silly",
			ClientID:   os.Getenv("TRAKT_ID"),
		}
		tmpl.Execute(w, data)
	}).Methods("GET")
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = "0.0.0.0:8000"
	}
	log.Print("Started on " + listen + "!")
	log.Fatal(http.ListenAndServe(listen, router))
}
