package api

import (
	"context"
	"net/http"
	"time"

	"github.com/etherlabsio/healthcheck"
)

func HealthCheckHandler() http.Handler {
	return healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker("storage", healthcheck.CheckerFunc(func(ctx context.Context) error {
			return storage.Ping(ctx)
		})),
	)
}
