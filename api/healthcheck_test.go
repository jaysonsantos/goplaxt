package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	var rr *httptest.ResponseRecorder

	r, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	storage = &MockSuccessStore{}
	rr = httptest.NewRecorder()
	http.Handler(HealthCheckHandler()).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
	assert.Equal(t, "{\"status\":\"OK\"}\n", rr.Body.String())

	storage = &MockFailStore{}
	rr = httptest.NewRecorder()
	http.Handler(HealthCheckHandler()).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Result().StatusCode)
	assert.Equal(t, "{\"status\":\"Service Unavailable\",\"errors\":{\"storage\":\"OH NO\"}}\n", rr.Body.String())
}
