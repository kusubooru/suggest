package teian

import (
	"sort"
	"strings"
	"time"
)

type AliasStatus int

const (
	AliasNew AliasStatus = iota
	AliasApproved
	AliasRejected
)

func (as AliasStatus) IsNew() bool      { return as == AliasNew }
func (as AliasStatus) IsApproved() bool { return as == AliasApproved }
func (as AliasStatus) IsRejected() bool { return as == AliasRejected }

// Alias represents an alias that a user can suggest to be created.
type Alias struct {
	ID       uint64
	Username string
	Old      string
	New      string
	Comment  string
	Created  time.Time
	Status   AliasStatus
}

// FmtCreated returns the creation time of the alias formatted as:
//
//     2006-01-02 15:04
func (a *Alias) FmtCreated() string {
	return a.Created.UTC().Format("2006-01-02 15:04")
}

// SearchAliasByID first sorts the slice of alias by id and then searches for
// the given id. If the id is found in the slice then it returns the index and
// the suggestion. If the id is not found then it returns index -1 and nil.
func SearchAliasByID(alias []*Alias, id uint64) (int, *Alias) {
	sort.Slice(alias, func(i, j int) bool { return alias[i].ID < alias[j].ID })
	i := sort.Search(len(alias), func(i int) bool { return alias[i].ID >= id })
	if i < len(alias) && alias[i].ID == id {
		return i, alias[i]
	}
	return -1, nil
}

// FilterAliasByUsername returns alias whose Username contains the provided
// username. It is used to filter a slice of alias by username.
func FilterAliasByUsername(alias []*Alias, username string) []*Alias {
	var f []*Alias
	for _, s := range alias {
		if strings.Contains(s.Username, username) {
			f = append(f, s)
		}
	}
	return f
}

// FilterAliasByComment returns alias whose Comment contains the provided text.
// It is used to search a slice of alias and return only those whose comment
// contains the given text.
func FilterAliasByComment(alias []*Alias, text string) []*Alias {
	var f []*Alias
	for _, s := range alias {
		if strings.Contains(s.Comment, text) {
			f = append(f, s)
		}
	}
	return f
}

func FilterAlias(alias []*Alias, filter string, filterFn func(int, string) bool) []*Alias {
	var f []*Alias
	for i := range alias {
		if filterFn(i, filter) {
			f = append(f, alias[i])
		}
	}
	return f
}
