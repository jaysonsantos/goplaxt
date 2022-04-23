package main

import (
	"context"
	"html/template"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/xanderstrike/goplaxt/api"
	"github.com/xanderstrike/goplaxt/lib/store"
	"github.com/xanderstrike/goplaxt/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

type traceIdPrint struct {
	h http.Handler
}

// ServeHTTP implements http.Handler
func (t *traceIdPrint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.Tracer.Start(r.Context(), "traceIdPrint")
	defer span.End()
	log.WithContext(ctx).WithField("traceId", span.SpanContext().TraceID().String()).Info("here you go")
	t.h.ServeHTTP(w, r)
}

func main() {
	ctx := context.Background()
	tracerShutdown, err := tracing.InitProvider(ctx)
	if err != nil {
		log.Fatalf("failed to initialize opentelemetry %v", err)
	}
	defer tracerShutdown()

	logger := log.WithContext(ctx)
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		parsedLogLevel, err := log.ParseLevel(logLevel)
		if err != nil {
			logger.WithField("logLevel", logLevel).Error("failed to parse log level")
			parsedLogLevel = log.InfoLevel
		}
		log.SetLevel(parsedLogLevel)
	}
	logger.WithField("logLevel", log.GetLevel().String()).Print("Started!")
	var storage store.Store
	if os.Getenv("POSTGRESQL_URL") != "" {
		storage = store.NewPostgresqlStore(store.NewPostgresqlClient(os.Getenv("POSTGRESQL_URL")))
		logger.Println("Using postgresql storage:", os.Getenv("POSTGRESQL_URL"))
	} else if os.Getenv("REDIS_URI") != "" {
		storage = store.NewRedisStore(store.NewRedisClient(os.Getenv("REDIS_URI"), os.Getenv("REDIS_PASSWORD")))
		logger.Println("Using redis storage:", os.Getenv("REDIS_URI"))
	} else {
		storage = store.NewDiskStore()
		logger.Println("Using disk storage:")
	}
	api.SetStore(storage)

	router := mux.NewRouter()
	router.Use(otelmux.Middleware("goplaxt"))
	router.Use(func(h http.Handler) http.Handler {
		return &traceIdPrint{h}
	})
	// Assumption: Behind a proper web server (nginx/traefik, etc) that removes/replaces trusted headers
	router.Use(handlers.ProxyHeaders)
	// which hostnames we are allowing
	// REDIRECT_URI = old legacy list
	// ALLOWED_HOSTNAMES = new accurate config variable
	// No env = all hostnames
	if os.Getenv("REDIRECT_URI") != "" {
		router.Use(api.AllowedHostsHandler(os.Getenv("REDIRECT_URI")))
	} else if os.Getenv("ALLOWED_HOSTNAMES") != "" {
		router.Use(api.AllowedHostsHandler(os.Getenv("ALLOWED_HOSTNAMES")))
	}
	router.HandleFunc("/authorize", api.Authorize).Methods("GET")
	router.HandleFunc("/api", api.ApiHandler).Methods("POST")
	router.Handle("/healthcheck", api.HealthCheckHandler()).Methods("GET")
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("static/index.html"))
		data := api.AuthorizePage{
			SelfRoot:   api.SelfRoot(r),
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
	logger.Print("Started on " + listen + "!")
	logger.Fatal(http.ListenAndServe(listen, router))
}
