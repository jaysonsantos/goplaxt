package store

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gravitational/trace"
)

type store interface {
	WriteUser(ctx context.Context, user User) error
}

// User object
type User struct {
	ID           string
	Username     string
	AccessToken  string
	RefreshToken string
	Updated      time.Time
	store        store
}

func uuid() string {
	f, _ := os.OpenFile("/dev/urandom", os.O_RDONLY, 0)
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	uuid := fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return uuid
}

// NewUser creates a new user object
func NewUser(ctx context.Context, username, accessToken, refreshToken string, store store) (*User, error) {
	id := uuid()
	user := User{
		ID:           id,
		Username:     username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Updated:      time.Now(),
		store:        store,
	}
	if err := user.save(ctx); err != nil {
		return nil, trace.Wrap(err)
	}
	return &user, nil
}

// UpdateUser updates an existing user object
func (user User) UpdateUser(ctx context.Context, accessToken, refreshToken string) error {
	user.AccessToken = accessToken
	user.RefreshToken = refreshToken
	user.Updated = time.Now()

	return trace.Wrap(user.save(ctx))
}

func (user User) save(ctx context.Context) error {
	return trace.Wrap(user.store.WriteUser(ctx, user))
}
