package store

import (
	"context"
)

// Store is the interface for All the store types
type Store interface {
	WriteUser(user User) error
	GetUser(id string) (*User, error)
	DeleteUser(id string) bool
	Ping(ctx context.Context) error
}

// Utils
func flatTransform(s string) []string { return []string{} }
