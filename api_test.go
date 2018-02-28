package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kusubooru/shimmie"
)

type mockShim struct {
	AutocompleteFn  func(q string, limit, offset int) ([]*shimmie.Autocomplete, error)
	GetPMsFn        func(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error)
	GetUserByNameFn func(username string) (*shimmie.User, error)
}

func (m *mockShim) Autocomplete(q string, limit, offset int) ([]*shimmie.Autocomplete, error) {
	return m.AutocompleteFn(q, limit, offset)
}

func (m *mockShim) GetPMs(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error) {
	return m.GetPMsFn(from, to, choice)
}

func (m *mockShim) GetUserByName(username string) (*shimmie.User, error) {
	return m.GetUserByNameFn(username)
}

func TestAPI_handleShowUnread(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	req = req.WithContext(shimmie.NewContextWithUser(context.Background(), &shimmie.User{Name: "zoe"}))
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
	h := api.handleShowUnread
	if err := h(w, req); err != nil {
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

func checkStatusCode(t *testing.T, err error, code int) {
	t.Helper()
	ae, ok := err.(*apiErr)
	if !ok {
		t.Error("error should be api error")
	}
	if got, want := ae.Code, code; got != want {
		t.Errorf("api error returned code %d, want %d", got, want)
	}
}

func TestAPI_handleShowUnread_dbError(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	req = req.WithContext(shimmie.NewContextWithUser(context.Background(), &shimmie.User{Name: "zoe"}))
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
	checkStatusCode(t, err, 500)
}

func TestAPI_handleShowUnread_noUser(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	api := &API{Shimmie: &mockShim{}}
	err := api.handleShowUnread(w, req)
	if err == nil {
		t.Fatal("show unread handler without cookie should return error")
	}
	checkStatusCode(t, err, 401)
}

func TestAPI_auth(t *testing.T) {
	zoePasswordHash := shimmie.PasswordHash("zoe", "zoe123")
	req := httptest.NewRequest("GET", "http://foo", nil)
	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "zoe"})
	ip := shimmie.GetOriginalIP(req)
	req.AddCookie(&http.Cookie{Name: "shm_session", Value: shimmie.CookieValue(zoePasswordHash, ip)})

	mock := &mockShim{}
	zoeUser := &shimmie.User{Name: "zoe", Pass: zoePasswordHash}
	mock.GetUserByNameFn = func(user string) (*shimmie.User, error) {
		return zoeUser, nil
	}
	api := &API{Shimmie: mock}
	h := api.auth(apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		u, ok := shimmie.FromContextGetUser(r.Context())
		if !ok {
			t.Fatalf("user not found in context")
		}

		got, want := u, zoeUser
		if !reflect.DeepEqual(got, want) {
			t.Errorf("user in context after auth = \nhave: %#v\nwant: %#v", got, want)
		}
		return nil
	}))
	w := httptest.NewRecorder()
	if err := h(w, req); err != nil {
		t.Fatalf("auth handler returned error: %v", err)
	}
}

func TestAPI_auth_invalidSession(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "foo"})
	req.AddCookie(&http.Cookie{Name: "shm_session", Value: "bar"})

	mock := &mockShim{}
	mock.GetUserByNameFn = func(user string) (*shimmie.User, error) {
		zoeUser := &shimmie.User{Name: "zoe", Pass: "4b1d"}
		return zoeUser, nil
	}
	api := &API{Shimmie: mock}
	h := api.auth(apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}))
	err := h(w, req)
	if err == nil {
		t.Fatalf("auth handler with invalid session should return error")
	}
	checkStatusCode(t, err, 401)
}

func TestAPI_auth_noUserCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	mock := &mockShim{}
	api := &API{Shimmie: mock}
	h := api.auth(apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}))
	err := h(w, req)
	if err == nil {
		t.Fatalf("auth handler with no user cookie should return error")
	}
	checkStatusCode(t, err, 401)
}

func TestAPI_auth_noSessionCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "foo"})

	mock := &mockShim{}
	api := &API{Shimmie: mock}
	h := api.auth(apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}))
	err := h(w, req)
	if err == nil {
		t.Fatalf("auth handler with no session cookie should return error")
	}
	checkStatusCode(t, err, 401)
}

func TestAPI_auth_dbError(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "foo"})
	req.AddCookie(&http.Cookie{Name: "shm_session", Value: "bar"})

	mock := &mockShim{}
	mock.GetUserByNameFn = func(user string) (*shimmie.User, error) {
		return nil, fmt.Errorf("db error")
	}
	api := &API{Shimmie: mock}
	h := api.auth(apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}))
	err := h(w, req)
	if err == nil {
		t.Fatalf("auth handler with db error should return error")
	}
	checkStatusCode(t, err, 500)
}

func TestAPI_auth_sqlNoRows(t *testing.T) {
	req := httptest.NewRequest("GET", "http://foo", nil)
	w := httptest.NewRecorder()

	req.AddCookie(&http.Cookie{Name: "shm_user", Value: "foo"})
	req.AddCookie(&http.Cookie{Name: "shm_session", Value: "bar"})

	mock := &mockShim{}
	mock.GetUserByNameFn = func(user string) (*shimmie.User, error) {
		return nil, sql.ErrNoRows
	}
	api := &API{Shimmie: mock}
	h := api.auth(apiHandler(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}))
	err := h(w, req)
	if err == nil {
		t.Fatalf("auth handler with sql no rows error should return error")
	}
	checkStatusCode(t, err, 401)
}
