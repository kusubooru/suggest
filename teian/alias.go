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

var aliasStatuses = [...]string{
	"New",
	"Approved",
	"Rejected",
}

func (as AliasStatus) String() string   { return aliasStatuses[as] }
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
