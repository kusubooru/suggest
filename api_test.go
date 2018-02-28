package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kusubooru/shimmie"
)

type mockShim struct {
	AutocompleteFn func(q string, limit, offset int) ([]*shimmie.Autocomplete, error)
	GetPMsFn       func(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error)
}

func (m *mockShim) Autocomplete(q string, limit, offset int) ([]*shimmie.Autocomplete, error) {
	return m.AutocompleteFn(q, limit, offset)
}

func (m *mockShim) GetPMs(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error) {
	return m.GetPMsFn(from, to, choice)
}

func TestAPI_handleShowUnread(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "zoe"})
	w := httptest.NewRecorder()

	mock := &mockShim{}
	mock.GetPMsFn = func(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error) {
		pms := []*shimmie.PM{
			{FromUser: "bob", ToUser: "zoe", IsRead: false},
			{FromUser: "ann", ToUser: "zoe", IsRead: false},
		}
		return pms, nil
	}
	api := &API{Shimmie: mock}
	if err := api.handleShowUnread(w, req); err != nil {
		t.Fatalf("show unread handler returned error: %v", err)
	}

	resp := w.Result()

	showUnread := new(ShowUnreadResp)
	if err := json.NewDecoder(resp.Body).Decode(showUnread); err != nil {
		t.Fatalf("decoding show unread response failed: %v", err)
	}
	got, want := showUnread, &ShowUnreadResp{User: "zoe", Unread: 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("show unread resp \nhave: %#v\nwant: %#v", got, want)
	}
}

func TestAPI_handleShowUnread_dbError(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "zoe"})
	w := httptest.NewRecorder()

	mock := &mockShim{}
	mock.GetPMsFn = func(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error) {
		return nil, fmt.Errorf("db error")
	}
	api := &API{Shimmie: mock}
	err := api.handleShowUnread(w, req)
	if err == nil {
		t.Fatalf("show unread handler with db error should return error")
	}
}

func TestAPI_handleShowUnread_noCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	api := &API{Shimmie: &mockShim{}}
	err := api.handleShowUnread(w, req)
	if err == nil {
		t.Fatal("show unread handler without cookie should return error")
	}
}
