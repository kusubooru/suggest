package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/teian/internal/mock"
	"github.com/kusubooru/teian/teian"
)

func TestApp_handleSubmit(t *testing.T) {
	s := &mock.SuggestionStore{}
	s.CreateFn = func(username string, sugg *teian.Suggestion) error { return nil }
	app := App{Suggestions: s}

	h := app.handleSubmit("/bad", "/login", "/success")

	type req struct {
		method string
		v      url.Values
		u      *shimmie.User
	}
	type resp struct {
		code     int
		location string
	}
	tests := []struct {
		req  req
		resp resp
	}{
		{req{"GET", nil, nil}, resp{405, ""}},
		{req{"POST", nil, nil}, resp{302, "/login"}},
		{req{"POST", nil, &shimmie.User{Name: "jun"}}, resp{302, "/bad"}},
		{req{"POST", url.Values{"text": {"blah"}}, nil}, resp{302, "/login"}},
		{req{"POST", url.Values{"text": {"blah"}}, &shimmie.User{Name: "jin"}}, resp{303, "/success"}},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		r := newRequest(tt.req.method, tt.req.v, tt.req.u)
		h.ServeHTTP(w, r)
		resp := w.Result()
		if got, want := resp.StatusCode, tt.resp.code; got != want {
			t.Errorf("req(%q, %v, %v) StatusCode = %d, want %d", tt.req.method, tt.req.v, tt.req.u, got, want)
		}
		if got, want := resp.Header.Get("Location"), tt.resp.location; got != want {
			t.Errorf("req(%q, %v, %v) Location = %q, want %q", tt.req.method, tt.req.v, tt.req.u, got, want)
		}
	}
}

func newRequest(method string, form url.Values, u *shimmie.User) *http.Request {
	var r *http.Request
	if len(form) == 0 {
		r = httptest.NewRequest(method, "/foo", nil)
	} else {
		r = httptest.NewRequest(method, "/foo", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if u != nil {
		r = r.WithContext(shimmie.NewContextWithUser(r.Context(), u))
	}
	return r
}

var discardLogger = log.New(ioutil.Discard, "", 0)

func TestApp_handleSubmit_createSuggestionFailure(t *testing.T) {
	s := &mock.SuggestionStore{}
	s.CreateFn = func(username string, sugg *teian.Suggestion) error { return fmt.Errorf("boom") }
	app := App{Log: discardLogger, Suggestions: s}

	h := app.handleSubmit("/bad", "/login", "/success")

	method := "POST"
	v := url.Values{"text": {"blah"}}
	u := &shimmie.User{Name: "jin"}
	r := newRequest(method, v, u)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()
	if got, want := resp.StatusCode, 500; got != want {
		t.Errorf("StatusCode = %d, want %d", got, want)
	}
}
