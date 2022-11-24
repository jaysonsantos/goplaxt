package store

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"strings"
	"time"

	"github.com/go-redis/redis/extra/redisotel/v9"
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
	if err := redisotel.InstrumentTracing(client); err != nil {
		panic(err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		panic(err)
	}

	_, err := client.Ping(context.Background()).Result()
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
	_, err := s.client.Ping(ctx).Result()
	return err
}

// WriteUser will write a user object to redis
func (s RedisStore) WriteUser(ctx context.Context, user User) error {
	data := make(map[string]interface{})
	data["username"] = user.Username
	data["access"] = user.AccessToken
	data["refresh"] = user.RefreshToken
	data["updated"] = user.Updated.Format("01-02-2006")
	status := s.client.HMSet(ctx, fmt.Sprintf("goplaxt:user:%s", user.ID), data)
	return trace.Wrap(status.Err())
}

// GetUser will load a user from redis
func (s RedisStore) GetUser(ctx context.Context, id string) (*User, error) {
	data, err := s.client.HGetAll(ctx, "goplaxt:user:"+id).Result()
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
func (s RedisStore) DeleteUser(ctx context.Context, id string) bool {
	return true
}
