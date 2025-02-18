package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/gravitational/trace"
)

// RedisStore is a storage engine that writes to redis
type RedisStore struct {
	client redis.Client
}

// NewRedisClient creates a new redis client object
func NewRedisClient(addr string, password string) redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	_, err := client.Ping().Result()
	// FIXME
	if err != nil {
		panic(err)
	}
	return *client
}

// NewRedisStore creates new store
func NewRedisStore(client redis.Client) RedisStore {
	return RedisStore{
		client: client,
	}
}

// Ping will check if the connection works right
func (s RedisStore) Ping(ctx context.Context) error {
	_, err := s.client.WithContext(ctx).Ping().Result()
	return err
}

// WriteUser will write a user object to redis
func (s RedisStore) WriteUser(user User) error {
	data := make(map[string]interface{})
	data["username"] = user.Username
	data["access"] = user.AccessToken
	data["refresh"] = user.RefreshToken
	data["updated"] = user.Updated.Format("01-02-2006")
	status := s.client.HMSet(fmt.Sprintf("goplaxt:user:%s", user.ID), data)
	return trace.Wrap(status.Err())
}

// GetUser will load a user from redis
func (s RedisStore) GetUser(id string) (*User, error) {
	data, err := s.client.HGetAll("goplaxt:user:" + id).Result()
	// FIXME - return err
	if err != nil {
		return nil, trace.Wrap(err)
	}
	updated, err := time.Parse("01-02-2006", data["updated"])
	// FIXME - return err
	if err != nil {
		return nil, trace.Wrap(err)
	}
	user := User{
		ID:           id,
		Username:     strings.ToLower(data["username"]),
		AccessToken:  data["access"],
		RefreshToken: data["refresh"],
		Updated:      updated,
		store:        s,
	}

	return &user, nil
}

// TODO: Not Implemented
func (s RedisStore) DeleteUser(id string) bool {
	return true
}
