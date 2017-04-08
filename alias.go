package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/teian/teian"
)

func (app *App) handleAliasDelete(w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}
	// get user from context
	_, ok := shimmie.FromContextGetUser(r.Context())
	if !ok {
		http.Redirect(w, r, app.Conf.LoginURL, http.StatusFound)
		return
	}
	idValue := r.PostFormValue("id")
	id, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil {
		http.Error(w, "not valid alias id", http.StatusBadRequest)
		return
	}

	type result struct {
		Err  error
		Msg  string
		Type string
	}

	if err := app.Suggestions.DeleteAlias(id); err != nil {
		app.render(w, submitTmpl, result{Err: err, Type: "error", Msg: "Could not delete alias"})
		return
	}

	http.Redirect(w, r, "/suggest/alias", http.StatusFound)
}

func (app *App) handleAliasNew(w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}
	// get user from context
	user, ok := shimmie.FromContextGetUser(r.Context())
	if !ok {
		http.Redirect(w, r, app.Conf.LoginURL, http.StatusFound)
		return
	}
	old := r.PostFormValue("old")
	new := r.PostFormValue("new")
	comment := r.PostFormValue("comment")
	// redirect if suggestion text is empty
	if len(strings.TrimSpace(old)) == 0 || len(strings.TrimSpace(new)) == 0 {
		http.Redirect(w, r, "/suggest/alias", http.StatusFound)
		return
	}

	type result struct {
		Err  error
		Msg  string
		Type string
	}

	// create and store suggestion
	err := app.Suggestions.NewAlias(&teian.Alias{Old: old, New: new, Comment: comment, Username: user.Name})
	if err != nil {
		app.render(w, submitTmpl, result{Err: err, Type: "error", Msg: submitFailureMessage})
	}

	http.Redirect(w, r, "/suggest/alias", http.StatusFound)
}

func (app *App) serveAliasNew(w http.ResponseWriter, r *http.Request) {
	_, ok := shimmie.FromContextGetUser(r.Context())
	if !ok {
		http.Redirect(w, r, app.Conf.LoginURL, http.StatusFound)
		return
	}
	app.render(w, aliasNewTmpl, nil)
}

func (app *App) handleAlias(w http.ResponseWriter, r *http.Request) {

	idstr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, err := strconv.ParseUint(idstr, 10, 64)
	if err != nil {
		http.Error(w, "not valid alias id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		r = r.WithContext(NewContextWithAliasID(r.Context(), id))
		app.serveAliasEdit(w, r)
		return
	case "POST":
		r = r.WithContext(NewContextWithAliasID(r.Context(), id))
		app.handleAliasUpdate(w, r)
		return
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

type contextKey int

const (
	aliasIDContextKey contextKey = iota
)

// FromContextGetAliasID gets alias id from context. If alias id does not exist
// in context, nil and false are returned instead.
func FromContextGetAliasID(ctx context.Context) (uint64, bool) {
	id, ok := ctx.Value(aliasIDContextKey).(uint64)
	return id, ok
}

// NewContextWithAliasID adds alias id to context.
func NewContextWithAliasID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, aliasIDContextKey, id)
}

func (app *App) handleAliasUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := FromContextGetAliasID(r.Context())
	if !ok {
		http.Redirect(w, r, "/suggest/alias", http.StatusFound)
		return
	}
	old := r.PostFormValue("old")
	new := r.PostFormValue("new")
	if strings.Contains(old, " ") || strings.Contains(new, " ") {
		http.Error(w, "a tag cannot contain spaces", http.StatusBadRequest)
		return
	}

	comment := r.PostFormValue("comment")

	statusStr := r.PostFormValue("status")
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s is not a valid alias status", statusStr), http.StatusBadRequest)
		return
	}

	a := &teian.Alias{
		Old:     old,
		New:     new,
		Comment: comment,
		Status:  teian.AliasStatus(status),
	}
	_, err = app.Suggestions.UpdateAlias(id, a)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/suggest/alias", http.StatusFound)
}

func (app *App) serveAliasEdit(w http.ResponseWriter, r *http.Request) {
	user, ok := shimmie.FromContextGetUser(r.Context())
	if !ok {
		http.Redirect(w, r, app.Conf.LoginURL, http.StatusFound)
		return
	}

	id, ok := FromContextGetAliasID(r.Context())
	if !ok {
		http.Redirect(w, r, "/suggest/alias", http.StatusFound)
		return
	}

	alias, err := app.Suggestions.GetAlias(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	if user.Admin == "Y" {
		app.render(w, aliasEditAdminTmpl, alias)
		return
	}
	if user.Name == alias.Username {
		app.render(w, aliasEditNormalTmpl, alias)
		return
	}
	app.render(w, aliasShowTmpl, alias)
}

func (app *App) serveAliasSearch(w http.ResponseWriter, r *http.Request) {
	//err := app.Suggestions.DeleteAllAlias()
	//if err != nil {
	//	http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
	//	return
	//}

	_, ok := shimmie.FromContextGetUser(r.Context())
	if !ok {
		http.Redirect(w, r, app.Conf.LoginURL, http.StatusFound)
		return
	}
	alias, err := app.Suggestions.AllAlias()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	username := r.FormValue("username")
	old := r.FormValue("old")
	new := r.FormValue("new")
	comment := r.FormValue("comment")
	order := r.FormValue("order")

	alias = filterAlias(alias, username, old, new, comment, order)
	app.render(w, aliasSearchTmpl, alias)
}

func filterAlias(alias []*teian.Alias, username, old, new, comment, order string) []*teian.Alias {
	if len(alias) == 0 || len(alias) == 1 {
		return alias
	}
	// filter
	if username != "" {
		alias = teian.FilterAlias(alias, username, func(i int, text string) bool { return strings.Contains(alias[i].Username, text) })
	}
	if comment != "" {
		alias = teian.FilterAlias(alias, comment, func(i int, text string) bool { return strings.Contains(alias[i].Comment, text) })
	}

	// order
	switch order {
	case "ua":
		sort.Slice(alias, func(i, j int) bool { return alias[i].Username < alias[j].Username })
	case "ud":
		sort.Slice(alias, func(i, j int) bool { return alias[i].Username > alias[j].Username })
	case "da":
		sort.Slice(alias, func(i, j int) bool { return alias[i].Created.Before(alias[j].Created) })
	case "dd":
		fallthrough
	default:
		sort.Slice(alias, func(i, j int) bool { return alias[i].Created.After(alias[j].Created) })
	}
	return alias
}
