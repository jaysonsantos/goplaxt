package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestLoadingUser(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))

	s.HSet("goplaxt:user:id123", "username", "halkeye")
	s.HSet("goplaxt:user:id123", "access", "access123")
	s.HSet("goplaxt:user:id123", "refresh", "refresh123")
	s.HSet("goplaxt:user:id123", "updated", "02-25-2019")

	expected, err := json.Marshal(&User{
		ID:           "id123",
		Username:     "halkeye",
		AccessToken:  "access123",
		RefreshToken: "refresh123",
		Updated:      time.Date(2019, 02, 25, 0, 0, 0, 0, time.UTC),
	})
	user, err := store.GetUser("id123")
	if err != nil {
		panic(err)
	}
	actual, err := json.Marshal(user)

	assert.EqualValues(t, string(expected), string(actual))
}

func TestSavingUser(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))
	originalUser := &User{
		ID:           "id123",
		Username:     "halkeye",
		AccessToken:  "access123",
		RefreshToken: "refresh123",
		Updated:      time.Date(2019, 02, 25, 0, 0, 0, 0, time.UTC),
		store:        store,
	}

	originalUser.save()

	assert.Equal(t, s.HGet("goplaxt:user:id123", "username"), "halkeye")
	assert.Equal(t, s.HGet("goplaxt:user:id123", "access"), "access123")
	assert.Equal(t, s.HGet("goplaxt:user:id123", "refresh"), "refresh123")
	assert.Equal(t, s.HGet("goplaxt:user:id123", "updated"), "02-25-2019")

	expected, _ := json.Marshal(originalUser)
	user, _ := store.GetUser("id123")
	actual, _ := json.Marshal(user)

	assert.EqualValues(t, string(expected), string(actual))
}

func TestPing(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))
	assert.Equal(t, store.Ping(context.TODO()), nil)
}
