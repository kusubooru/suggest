package teian

import (
	"sort"
	"strings"
	"time"
)

// User represents a user of the application.
type User struct {
	ID       int
	Name     string
	Pass     string
	JoinDate *time.Time
	Admin    string
	Email    string
	Class    string
}

// Conf holds configuration values that the program needs.
type Conf struct {
	Title       string
	AnalyticsID string
	Description string
	Keywords    string
	WriteMsg    string
}

// SiteTitle returns the Title capitalized.
func (c Conf) SiteTitle() string {
	return strings.Title(c.Title)
}

// SuggStore describes all the operations that need to access a storage for the
// suggestions.
type SuggStore interface {
	// Create creates a new suggestion for a user.
	Create(username string, sugg *Sugg) error
	// GetSugg gets all the suggestions created by a user.
	GetSugg(username string) ([]Sugg, error)
	// GetSugg gets all the suggestions.
	GetAllSugg() ([]Sugg, error)
	// Delete deletes a user's suggestion.
	Delete(username string, id uint64) error

	// Close releases all database resources.
	Close()
}

// Sugg represents a suggestion that a user can create.
type Sugg struct {
	ID       uint64
	Username string
	Text     string
	Created  time.Time
}

// FmtCreated returns the creation time of the suggestion formatted as:
//
//     Mon 02 Jan 2006 15:04:05 MST
func (s *Sugg) FmtCreated() string {
	return s.Created.UTC().Format("Mon 02 Jan 2006 15:04:05 MST")
}

// ByDate helps to sort suggestions by creation time.
type ByDate []Sugg

func (s ByDate) Len() int           { return len(s) }
func (s ByDate) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByDate) Less(i, j int) bool { return s[i].Created.Before(s[j].Created) }

// ByUser helps to sort suggestions by username.
type ByUser []Sugg

func (s ByUser) Len() int           { return len(s) }
func (s ByUser) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByUser) Less(i, j int) bool { return s[i].Username < s[j].Username }

// ByID helps to sort suggestions by ID.
type ByID []Sugg

func (s ByID) Len() int           { return len(s) }
func (s ByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByID) Less(i, j int) bool { return s[i].ID < s[j].ID }

// FindByID first sorts the slice of suggestions by ID and then searches for
// the given id. If the id is found in the slice then it returns the index and
// the suggestion. If the id is not found then it returns index -1 and nil.
func FindByID(suggs []Sugg, id uint64) (int, *Sugg) {
	sort.Sort(ByID(suggs))
	i := sort.Search(len(suggs), func(i int) bool { return suggs[i].ID >= id })
	if i < len(suggs) && suggs[i].ID == id {
		return i, &suggs[i]
	}
	return -1, nil
}

// FilterByUser returns suggestions whose Username contains the provided
// username. It is used to filter a slice of suggestions by username.
func FilterByUser(suggs []Sugg, username string) []Sugg {
	var f []Sugg
	for _, s := range suggs {
		if strings.Contains(s.Username, username) {
			f = append(f, s)
		}
	}
	return f
}

// FilterByText returns suggestions whose Text contains the provided text. It
// is used to search a slice of suggestions and return only those containing
// the given text.
func FilterByText(suggs []Sugg, text string) []Sugg {
	var f []Sugg
	for _, s := range suggs {
		if strings.Contains(s.Text, text) {
			f = append(f, s)
		}
	}
	return f
}
