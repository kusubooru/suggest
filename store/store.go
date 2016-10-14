package store

import "github.com/kusubooru/teian/teian"

// Store describes all the operations that need to access database storage.
type Store interface {
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
