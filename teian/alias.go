package teian

import "time"

// Alias represents an alias that a user can suggest to be created.
type Alias struct {
	ID       uint64
	Username string
	Old      string
	New      string
	Comment  string
	Created  time.Time
}
