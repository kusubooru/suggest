package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kusubooru/shimmie"
)

type internal interface {
	Internal() bool
}

// IsInternal returns true if err is internal.
func IsInternal(err error) bool {
	e, ok := err.(internal)
	return ok && e.Internal()
}

type apiErr struct {
	err     error  `json:"-"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *apiErr) Error() string  { return fmt.Sprintf("%d %v: %v", e.Code, e.Message, e.err) }
func (e *apiErr) Internal() bool { return e.Code == http.StatusInternalServerError }

func E(err error, message string, code int) error {
	return &apiErr{err: err, Message: message, Code: code}
}

type apiHandler func(http.ResponseWriter, *http.Request) error

func (fn apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		switch e := err.(type) {
		case *apiErr:
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(e.Code)
			if IsInternal(err) {
				log.Println(err)
			}
			if encErr := json.NewEncoder(w).Encode(e); encErr != nil {
				http.Error(w, encErr.Error(), http.StatusInternalServerError)
			}
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type shim interface {
	Autocomplete(q string, limit, offset int) ([]*shimmie.Autocomplete, error)
	GetPMs(from, to string, choice shimmie.PMChoice) ([]*shimmie.PM, error)
	GetUserByName(username string) (*shimmie.User, error)
}

type API struct {
	Shimmie shim
}

type Tag struct {
	Name  string `json:"name"`
	Old   string `json:"old"`
	Count int    `json:"count"`
}

func (api *API) handleAutocomplete(w http.ResponseWriter, r *http.Request) error {
	q := r.FormValue("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte("{}"))
		return nil
	}
	if len(q) < 2 {
		return E(nil, "need at least 2 characters", http.StatusBadRequest)
	}

	tags, err := api.Shimmie.Autocomplete(q, 5, 0)
	if err != nil {
		return E(err, "Autocomplete failed.", http.StatusInternalServerError)
	}

	if err := json.NewEncoder(w).Encode(tags); err != nil {
		return E(err, "Could not encode autocomplete response.", http.StatusInternalServerError)
	}
	return nil
}

type ShowUnreadResp struct {
	User   string `json:"user"`
	Unread int    `json:"unread"`
}

func (api *API) handleShowUnread(w http.ResponseWriter, r *http.Request) error {
	user, ok := shimmie.FromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, "Unknown user.", http.StatusUnauthorized)
	}

	username := user.Name
	pms, err := api.Shimmie.GetPMs("", username, shimmie.PMUnread)
	if err != nil {
		return E(err, "Retrieving private messages failed.", http.StatusInternalServerError)
	}

	resp := ShowUnreadResp{
		User:   username,
		Unread: len(pms),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return E(err, "Could not encode unread private messages.", http.StatusInternalServerError)
	}
	return nil
}

func (api *API) auth(h apiHandler) apiHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		usernameCookie, err := r.Cookie("shm_user")
		if err != nil || usernameCookie.Value == "" {
			return E(err, "Unknown user.", http.StatusUnauthorized)
		}

		sessionCookie, err := r.Cookie("shm_session")
		if err != nil {
			return E(err, "No session.", http.StatusUnauthorized)
		}
		username := usernameCookie.Value
		user, err := api.Shimmie.GetUserByName(username)
		switch err {
		case sql.ErrNoRows:
			return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		case nil:
		default:
			return E(err, "Could not retrieve user.", http.StatusInternalServerError)
		}

		passwordHash := user.Pass
		userIP := shimmie.GetOriginalIP(r)
		sessionCookieValue := shimmie.CookieValue(passwordHash, userIP)
		if sessionCookieValue != sessionCookie.Value {
			return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}

		ctx := shimmie.NewContextWithUser(r.Context(), user)
		h.ServeHTTP(w, r.WithContext(ctx))
		return nil
	}
}
