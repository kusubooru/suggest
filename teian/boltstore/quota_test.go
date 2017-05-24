package boltstore

import (
	"testing"

	"github.com/kusubooru/teian/teian"
)

func TestQuota(t *testing.T) {
	store, f := setup()
	defer teardown(store, f)

	username := "john"

	// add 5 out of 10 MB quota
	_, err := store.CheckQuota(username, teian.Quota(5<<20))
	if err != nil {
		t.Fatal("store.CheckQuota failed:", err)
	}
	// add 5 + 2 out of 10 MB quota
	remain, err := store.CheckQuota(username, teian.Quota(2<<20))
	if err != nil {
		t.Fatal("store.CheckQuota failed:", err)
	}
	// expect remain to be 3 MB
	got, want := int64(remain), int64(3<<20)
	if got != want {
		t.Fatalf("store.CheckQuota should return remain %v, got %v", got, want)
	}
	// try to add 4 MB more while only 3 MB remain
	_, err = store.CheckQuota(username, teian.Quota(4<<20))
	if err != teian.ErrOverQuota {
		t.Error("store.CheckQuota expected to return ErrOverQuota error, got:", err)
	}

	// clean quota
	if err := store.CleanQuota(); err != nil {
		t.Fatal("store.CleanQuota failed:", err)
	}

	// Add 10 out of 10 MB quota which should succeed after the clean.
	remain, err = store.CheckQuota(username, teian.Quota(10<<20))
	if err != nil {
		t.Fatal("store.CheckQuota failed:", err)
	}
	// expect remain to be 0 MB
	got, want = int64(remain), int64(0<<20)
	if got != want {
		t.Fatalf("store.CheckQuota should return remain %v, got %v", got, want)
	}
}
