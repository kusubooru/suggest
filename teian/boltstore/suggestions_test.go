package boltstore

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/kusubooru/teian/teian"
)

const testQuota = 10 << 20 // 10 MB

func setup() (*Boltstore, *os.File) {
	f, err := ioutil.TempFile("", "teian_boltdb_tmpfile_")
	if err != nil {
		log.Fatal("could not create boltdb temp file for tests:", err)
	}
	return NewSuggestionStore(f.Name(), testQuota), f
}

func teardown(store *Boltstore, tmpfile *os.File) {
	store.Close()
	if err := os.Remove(tmpfile.Name()); err != nil {
		log.Println("could not remove boltdb temp file:", err)
	}
}

func TestCreate(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	username := "john"
	text := "my first suggestion"

	err := store.Create(username, &teian.Suggestion{Text: text})
	if err != nil {
		t.Error("store.Create failed:", err)
	}
	out, err := store.OfUser(username)
	if err != nil {
		t.Fatal("store.All failed:", err)
	}
	if len(out) != 1 {
		t.Fatal("store.All should return result with length of 1")
	}
	got := out[0]
	want := teian.Suggestion{ID: 1, Username: username, Text: text, Created: got.Created}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("store.Create(%q, &teian.Suggestion{Test: %q}) lead to \n%#v, want \n%#v", username, text, got, want)
	}
}

func TestAll_Delete(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	username1 := "john"
	text1 := username1 + "'s suggestion text"
	for i := 0; i < 10; i++ {
		err := store.Create(username1, &teian.Suggestion{Text: fmt.Sprintf("%s #%d", text1, i)})
		if err != nil {
			t.Error("store.Create failed:", err)
		}
	}
	username2 := "mary"
	text2 := username2 + "'s suggestion text "
	for i := 0; i < 10; i++ {
		err := store.Create(username2, &teian.Suggestion{Text: fmt.Sprintf("%s #%d", text2, i)})
		if err != nil {
			t.Error("store.Create failed:", err)
		}
	}

	out, err := store.All()
	if err != nil {
		t.Fatal("store.All failed:", err)
	}
	if len(out) != 20 {
		t.Fatal("store.All should return result with length of 20")
	}

	// test delete
	for i := 1; i <= 10; i++ {
		err := store.Delete(username1, uint64(i))
		if err != nil {
			t.Errorf("store.Delete(%q, %d) failed: %v", username1, i, err)
		}
	}
	out, err = store.All()
	if err != nil {
		t.Fatal("store.All failed:", err)
	}
	if len(out) != 10 {
		t.Fatal("store.All should return result with length of 10")
	}
}
