package store

import (
	"fmt"
	"os"
	"time"

	"github.com/gravitational/trace"
)

type store interface {
	WriteUser(user User) error
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
func NewUser(username, accessToken, refreshToken string, store store) (*User, error) {
	id := uuid()
	user := User{
		ID:           id,
		Username:     username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Updated:      time.Now(),
		store:        store,
	}
	if err := user.save(); err != nil {
		return nil, trace.Wrap(err)
	}
	return &user, nil
}

// UpdateUser updates an existing user object
func (user User) UpdateUser(accessToken, refreshToken string) {
	user.AccessToken = accessToken
	user.RefreshToken = refreshToken
	user.Updated = time.Now()

	user.save()
}

func (user User) save() error {
	return user.store.WriteUser(user)
}
