package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xanderstrike/goplaxt/lib/store"
)

type MockSuccessStore struct{}

func (s MockSuccessStore) Ping(ctx context.Context) error                       { return nil }
func (s MockSuccessStore) WriteUser(ctx context.Context, user store.User) error { return nil }
func (s MockSuccessStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	return nil, nil
}
func (s MockSuccessStore) DeleteUser(ctx context.Context, id string) bool { return true }

type MockFailStore struct{}

func (s MockFailStore) Ping(ctx context.Context) error { return errors.New("OH NO") }
func (s MockFailStore) WriteUser(ctx context.Context, user store.User) error {
	panic(errors.New("OH NO"))
}
func (s MockFailStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	panic(errors.New("OH NO"))
}
func (s MockFailStore) DeleteUser(ctx context.Context, id string) bool { return false }

func TestSelfRoot(t *testing.T) {
	var (
		r   *http.Request
		err error
	)

	// Test Default
	r, err = http.NewRequest("GET", "/authorize", nil)
	r.Host = "foo.bar"
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "http://foo.bar", SelfRoot(r))

	// Test Manual forwarded proto
	r, err = http.NewRequest("GET", "/validate", nil)
	r.Host = "foo.bar"
	r.Header.Set("X-Forwarded-Proto", "https")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "https://foo.bar", SelfRoot(r))

	// Test ProxyHeader handler
	rr := httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/validate", nil)
	require.NoError(t, err)
	r.Header.Set("X-Forwarded-Host", "foo.bar")
	r.Header.Set("X-Forwarded-Proto", "https")
	handlers.ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, "https://foo.bar", SelfRoot(r))
}

func TestAllowedHostsHandler_single_hostname(t *testing.T) {
	f := AllowedHostsHandler("foo.bar")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "foo.bar"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

func TestAllowedHostsHandler_multiple_hostnames(t *testing.T) {
	f := AllowedHostsHandler("foo.bar, bar.foo")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "bar.foo"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

func TestAllowedHostsHandler_mismatch_hostname(t *testing.T) {
	f := AllowedHostsHandler("unknown.host")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "known.host"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestAllowedHostsHandler_alwaysAllowHealthcheck(t *testing.T) {
	storage = &MockSuccessStore{}
	f := AllowedHostsHandler("unknown.host")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "known.host"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}
