package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/shimmie/store"
	"github.com/kusubooru/teian/teian"
	"github.com/kusubooru/teian/teian/boltstore"
)

//go:generate go run generate/templates.go

var (
	theVersion = "devel"
	versionRx  = regexp.MustCompile(`\d.*`)
	fns        = template.FuncMap{
		"join": strings.Join,
		"filterEmpty": func(s, filter string) string {
			if s == "" {
				return filter
			}
			return s
		},
		"formatTime": func(t time.Time) string {
			return t.Format("January 2, 2006; 15:04")
		},
		"printv": func(version string) string {
			// If version starts with a digit, add 'v'.
			if versionRx.Match([]byte(version)) {
				version = "v" + version
			}
			return version
		},
	}
)

const (
	description = `
  A service that allows users to submit suggestions.
`
	writeMessage         = `Do you have a suggestion on how to improve the site? Write it here!`
	submitSuccessMessage = "Your suggestion has been submitted. Thank you for your feedback!"
	submitFailureMessage = "Something broke! :'( Our developers were notified."
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, description)
	fmt.Fprintf(os.Stderr, "Options:\n\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
}

func main() {
	var (
		httpAddr  = flag.String("http", "localhost:8080", "HTTP listen address")
		dbDriver  = flag.String("dbdriver", "mysql", "database driver")
		dbConfig  = flag.String("dbconfig", "", "username:password@(host:port)/database?parseTime=true")
		boltFile  = flag.String("boltfile", "teian.db", "BoltDB database file to store suggestions")
		loginURL  = flag.String("loginurl", "/suggest/login", "login URL path to redirect to")
		writeMsg  = flag.String("writemsg", writeMessage, "message that appears on new suggestion screen")
		version   = flag.Bool("v", false, "print program version")
		certFile  = flag.String("tlscert", "", "TLS public key in PEM format.  Must be used together with -tlskey")
		keyFile   = flag.String("tlskey", "", "TLS private key in PEM format. Must be used together with -tlscert")
		imagePath = flag.String("imagepath", "", "path where images are stored")
		thumbPath = flag.String("thumbpath", "", "path where image thumbnails are stored")
		// Set after flag parsing based on certFile & keyFile.
		useTLS bool
	)
	flag.Usage = usage
	flag.Parse()
	useTLS = *certFile != "" && *keyFile != ""

	if *version {
		fmt.Printf("%s v%s (runtime: %s)\n", os.Args[0], theVersion, runtime.Version())
		os.Exit(0)
	}

	// open store with new database connection and create new Shimmie
	shim := shimmie.New(*imagePath, *thumbPath, store.Open(*dbDriver, *dbConfig))

	// get common conf
	common, cerr := shim.Store.GetCommon()
	if cerr != nil {
		log.Fatalln("could not get common conf:", cerr)
	}

	// create suggestion store
	suggStore := boltstore.NewSuggestionStore(*boltFile)
	closeStoreOnSignal(suggStore)

	app := &App{
		Shimmie:     shim,
		Suggestions: suggStore,
		Conf: &teian.Conf{
			Title:       common.Title,
			AnalyticsID: common.AnalyticsID,
			Description: common.Description,
			Keywords:    common.Keywords,
			WriteMsg:    *writeMsg,
			Version:     theVersion,
			LoginURL:    *loginURL,
		},
	}

	http.Handle("/suggest", shim.Auth(app.serveIndex, *loginURL))
	http.Handle("/suggest/admin", shim.Auth(app.serveAdmin, *loginURL))
	http.Handle("/suggest/admin/delete", shim.Auth(app.handleDelete, *loginURL))
	http.Handle("/suggest/submit", shim.Auth(app.handleSubmit, *loginURL))
	http.Handle("/suggest/login", http.HandlerFunc(app.serveLogin))
	http.Handle("/suggest/login/submit", http.HandlerFunc(app.handleLogin))
	http.Handle("/suggest/logout", http.HandlerFunc(handleLogout))
	http.Handle("/suggest/alias", shim.Auth(app.serveAliasSearch, *loginURL))
	http.Handle("/suggest/alias/", shim.Auth(app.handleAlias, *loginURL))
	http.Handle("/suggest/alias/delete", shim.Auth(app.handleAliasDelete, *loginURL))
	http.Handle("/suggest/alias/new", shim.Auth(app.serveAliasNew, *loginURL))
	http.Handle("/suggest/alias/new/submit", shim.Auth(app.handleAliasNew, *loginURL))

	if useTLS {
		if err := http.ListenAndServeTLS(*httpAddr, *certFile, *keyFile, nil); err != nil {
			log.Fatalf("Could not start listening (TLS) on %v: %v", *httpAddr, err)
		}
	} else {
		if err := http.ListenAndServe(*httpAddr, nil); err != nil {
			log.Fatalf("Could not start listening on %v: %v", *httpAddr, err)
		}
	}
}

func closeStoreOnSignal(s teian.SuggestionStore) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		for sig := range c {
			log.Printf("%v signal received, releasing database resources and exiting...", sig)
			s.Close()
			os.Exit(1)
		}
	}()
}

// App represents the Teian application and holds its dependencies.
type App struct {
	Suggestions teian.SuggestionStore
	Conf        *teian.Conf
	Shimmie     *shimmie.Shimmie
}

func (app *App) serveIndex(w http.ResponseWriter, r *http.Request) {
	app.render(w, suggestionTmpl, nil)
}

func (app *App) serveLogin(w http.ResponseWriter, r *http.Request) {
	app.render(w, loginTmpl, nil)
}

func (app *App) serveAdmin(w http.ResponseWriter, r *http.Request) {
	user, ok := shimmie.FromContextGetUser(r.Context())
	if !ok || user.Admin != "Y" {
		http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
		return
	}
	suggs, err := app.Suggestions.All()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
	}

	u := r.FormValue("u")
	t := r.FormValue("t")
	o := r.FormValue("o")

	suggs = filterSuggestions(suggs, u, t, o)
	app.render(w, adminTmpl, suggs)
}

func filterSuggestions(suggs []teian.Suggestion, username, text, order string) []teian.Suggestion {
	if len(suggs) == 0 || len(suggs) == 1 {
		return suggs
	}
	// filter
	if username != "" {
		suggs = teian.FilterByUser(suggs, username)
	}
	if text != "" {
		suggs = teian.FilterByText(suggs, text)
	}

	// order
	switch order {
	case "ua":
		sort.Sort(teian.ByUser(suggs))
	case "ud":
		sort.Sort(sort.Reverse(teian.ByUser(suggs)))
	case "da":
		sort.Sort(teian.ByDate(suggs))
	case "dd":
		fallthrough
	default:
		sort.Sort(sort.Reverse(teian.ByDate(suggs)))
	}
	return suggs
}

func (app *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}
	user, ok := shimmie.FromContextGetUser(r.Context())
	if !ok || user.Admin != "Y" {
		http.Error(w, "You are not authorized to perform this action.", http.StatusUnauthorized)
		return
	}
	idValue := r.PostFormValue("id")
	username := r.PostFormValue("username")
	if idValue == "" || username == "" {
		http.Error(w, "id and username must be present", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad id provided: %v", err), http.StatusBadRequest)
		return
	}
	err = app.Suggestions.Delete(username, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("delete suggestion failed: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/suggest/admin", http.StatusFound)
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	user, err := app.Shimmie.GetUser(username)
	if err != nil {
		if err == sql.ErrNoRows {
			app.render(w, loginTmpl, "User does not exist.")
			return
		}
		msg := fmt.Sprintf("get user %q failed: %v", username, err.Error())
		log.Print(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	passwordHash := shimmie.PasswordHash(username, password)
	if user.Pass != passwordHash {
		app.render(w, loginTmpl, "Username and password do not match.")
		return
	}
	addr := strings.Split(r.RemoteAddr, ":")[0]
	cookieValue := shimmie.CookieValue(passwordHash, addr)
	shimmie.SetCookie(w, "shm_user", username)
	shimmie.SetCookie(w, "shm_session", cookieValue)
	http.Redirect(w, r, "/suggest", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Use POST to logout.", http.StatusMethodNotAllowed)
		return
	}
	shimmie.SetCookie(w, "shm_user", "")
	shimmie.SetCookie(w, "shm_session", "")
	http.Redirect(w, r, "/suggest", http.StatusFound)
}

func (app *App) handleSubmit(w http.ResponseWriter, r *http.Request) {
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
	text := r.PostFormValue("text")
	// redirect if suggestion text is empty
	if len(strings.TrimSpace(text)) == 0 {
		http.Redirect(w, r, "/suggest", http.StatusFound)
		return
	}

	type result struct {
		Err  error
		Msg  string
		Type string
	}

	// create and store suggestion
	err := app.Suggestions.Create(user.Name, &teian.Suggestion{Text: text})
	if err != nil {
		app.render(w, submitTmpl, result{Err: err, Type: "error", Msg: submitFailureMessage})
	}

	app.render(w, submitTmpl, result{Type: "success", Msg: submitSuccessMessage})

}

func render(w http.ResponseWriter, t *template.Template, data interface{}) {
	if err := t.Execute(w, data); err != nil {
		msg := fmt.Sprintln("could not render template:", err)
		log.Print(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}

func (app *App) render(w http.ResponseWriter, t *template.Template, data interface{}) {
	render(w, t, struct {
		Data interface{}
		Conf *teian.Conf
	}{
		Data: data,
		Conf: app.Conf,
	})
}
