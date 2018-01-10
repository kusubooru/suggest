package main

import (
	"encoding/base64"
	"net/http"
	"time"
)

// These are not currently used.

func SetFlash(w http.ResponseWriter, key string, value []byte) {
	c := &http.Cookie{Name: key, Value: base64.URLEncoding.EncodeToString(value)}
	http.SetCookie(w, c)
}

func Flash(w http.ResponseWriter, r *http.Request, name string) ([]byte, error) {
	c, err := r.Cookie(name)
	switch err {
	case http.ErrNoCookie:
		return nil, nil
	case nil:
	default:
		return nil, err
	}
	value, err := base64.URLEncoding.DecodeString(c.Value)
	if err != nil {
		return nil, err
	}
	c = &http.Cookie{Name: name, MaxAge: -1, Expires: time.Now().Add(-100 * time.Hour)}
	http.SetCookie(w, c)
	return value, nil
}
