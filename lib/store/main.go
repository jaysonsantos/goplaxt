package store

import (
	"context"
)

// Store is the interface for All the store types
type Store interface {
	WriteUser(ctx context.Context, user User) error
	GetUser(ctx context.Context, id string) (*User, error)
	DeleteUser(ctx context.Context, id string) bool
	Ping(ctx context.Context) error
}

// Utils
func flatTransform(s string) []string { return []string{} }
