package store

import (
	"errors"

	"github.com/kusubooru/teian/teian"
	"golang.org/x/net/context"
)

// Store describes all the operations that need to access database storage.
type Store interface {
	// GetUser gets a user by unique username.
	GetUser(username string) (*teian.User, error)
}

// GetUser gets a user by unique username.
func GetUser(ctx context.Context, username string) (*teian.User, error) {
	s, ok := FromContext(ctx)
	if !ok {
		return nil, errors.New("no store in context")
	}
	return s.GetUser(username)
}

const key = "store"

// NewContext returns a new Context carrying store.
func NewContext(ctx context.Context, store Store) context.Context {
	return context.WithValue(ctx, key, store)
}

// FromContext extracts the store from ctx, if present.
func FromContext(ctx context.Context) (Store, bool) {
	// ctx.Value returns nil if ctx has no value for the key;
	// the Store type assertion returns ok=false for nil.
	s, ok := ctx.Value(key).(Store)
	return s, ok
}
