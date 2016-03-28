package teian

import "time"

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
