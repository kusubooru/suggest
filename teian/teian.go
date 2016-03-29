package teian

import (
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

// Sugg represents a suggestion that a user can create.
type Sugg struct {
	ID       uint64
	Username string
	Text     string
	Created  time.Time
}

func (s *Sugg) FmtCreated() string {
	return s.Created.UTC().Format("Mon 02 Jan 2006 15:04:05 MST")
}

type ByDate []Sugg

func (s ByDate) Len() int           { return len(s) }
func (s ByDate) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByDate) Less(i, j int) bool { return s[i].Created.Before(s[j].Created) }

type ByUser []Sugg

func (s ByUser) Len() int           { return len(s) }
func (s ByUser) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByUser) Less(i, j int) bool { return s[i].Username < s[j].Username }

type ByID []Sugg

func (s ByID) Len() int           { return len(s) }
func (s ByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByID) Less(i, j int) bool { return s[i].ID < s[j].ID }

func FilterByUser(suggs []Sugg, username string) []Sugg {
	var f []Sugg
	for _, s := range suggs {
		if strings.Contains(s.Username, username) {
			f = append(f, s)
		}
	}
	return f
}

func FilterByText(suggs []Sugg, text string) []Sugg {
	var f []Sugg
	for _, s := range suggs {
		if strings.Contains(s.Text, text) {
			f = append(f, s)
		}
	}
	return f
}
