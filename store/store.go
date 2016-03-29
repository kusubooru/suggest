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

	// CreateSugg creates a new suggestion for a user.
	CreateSugg(username string, sugg *teian.Sugg) error
	// GetSugg gets all the suggestions created by a user.
	GetSugg(username string) ([]teian.Sugg, error)
	// GetSugg gets all the suggestions.
	GetAllSugg() ([]teian.Sugg, error)
	// DeleteSugg deletes a user's suggestion.
	DeleteSugg(username string, id uint64) error

	// Close releases all database resources.
	Close()
}

// GetUser gets a user by unique username.
func GetUser(ctx context.Context, username string) (*teian.User, error) {
	s, ok := FromContext(ctx)
	if !ok {
		return nil, errors.New("no store in context")
	}
	return s.GetUser(username)
}

// CreateSugg creates a new suggestion.
func CreateSugg(ctx context.Context, username string, sugg *teian.Sugg) error {
	s, ok := FromContext(ctx)
	if !ok {
		return errors.New("no store in context")
	}
	return s.CreateSugg(username, sugg)
}

// GetSugg gets all the suggestions created by a user.
func GetSugg(ctx context.Context, username string) ([]teian.Sugg, error) {
	s, ok := FromContext(ctx)
	if !ok {
		return nil, errors.New("no store in context")
	}
	return s.GetSugg(username)
}

// GetAllSugg gets all the suggestions.
func GetAllSugg(ctx context.Context) ([]teian.Sugg, error) {
	s, ok := FromContext(ctx)
	if !ok {
		return nil, errors.New("no store in context")
	}
	return s.GetAllSugg()
}

// DeleteSugg deletes a user's suggestion.
func DeleteSugg(ctx context.Context, username string, id uint64) error {
	s, ok := FromContext(ctx)
	if !ok {
		return errors.New("no store in context")
	}
	return s.DeleteSugg(username, id)
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
