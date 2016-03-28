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
	ID       int64
	Username string
	Text     string
	Created  time.Time
}
