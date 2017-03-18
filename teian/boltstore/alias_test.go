package boltstore

import (
	"reflect"
	"testing"

	"github.com/kusubooru/teian/teian"
)

func TestNewAlias(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	var (
		username = "john"
		old      = "old_tag"
		new      = "new_tag"
		comment  = "plz add this"
	)

	in := &teian.Alias{Old: old, New: new, Comment: comment}
	err := store.NewAlias(username, in)
	if err != nil {
		t.Error("store.NewAlias failed:", err)
	}
	out, err := store.GetAliasOfUser(username)
	if err != nil {
		t.Fatal("store.GetAliasOfUser failed:", err)
	}
	if len(out) != 1 {
		t.Fatal("store.GetAliasOfUser should return result with length of 1")
	}
	got := out[0]
	want := &teian.Alias{ID: 1, Username: username, Old: old, New: new, Comment: comment, Created: got.Created}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.NewAlias(%q, %q) = \nhave: %#v\n want:%#v", username, in, got, want)
	}
}
