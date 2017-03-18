package boltstore

import (
	"fmt"
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

func TestAllAlias_DeleteAlias(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	var (
		username1 = "john"
		comment1  = username1 + "'s suggestion text"
		old1      = "old_tag1"
		new1      = "new_tag1"
	)
	for i := 0; i < 10; i++ {
		err := store.NewAlias(username1, &teian.Alias{Old: old1, New: new1, Comment: fmt.Sprintf("%s #%d", comment1, i)})
		if err != nil {
			t.Error("store.NewAlias failed:", err)
		}
	}
	var (
		username2 = "mary"
		comment2  = username2 + "'s suggestion text "
		old2      = "old_tag2"
		new2      = "new_tag2"
	)
	for i := 0; i < 10; i++ {
		err := store.NewAlias(username2, &teian.Alias{Old: old2, New: new2, Comment: fmt.Sprintf("%s #%d", comment2, i)})
		if err != nil {
			t.Error("store.NewAlias failed:", err)
		}
	}

	out, err := store.AllAlias()
	if err != nil {
		t.Fatal("store.AllAlias failed:", err)
	}
	if len(out) != 20 {
		t.Fatal("store.AllAlias should return result with length of 20")
	}

	// test delete
	for i := 1; i <= 10; i++ {
		if err = store.DeleteAlias(username1, uint64(i)); err != nil {
			t.Errorf("store.DeleteAlias(%q, %d) failed: %v", username1, i, err)
		}
	}
	out, err = store.AllAlias()
	if err != nil {
		t.Fatal("store.AllAlias failed:", err)
	}
	if len(out) != 10 {
		t.Fatal("store.AllAlias should return result with length of 10")
	}
}
