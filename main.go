package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/net/context"

	"github.com/kusubooru/teian/shimmie"
	"github.com/kusubooru/teian/store"
	"github.com/kusubooru/teian/store/datastore"
	"github.com/kusubooru/teian/teian"
)

var (
	httpAddr = flag.String("http", "localhost:8080", "HTTP listen address")
	dbDriver = flag.String("driver", "mysql", "Database driver")
	dbConfig = flag.String("config", "", "username:password@(host:port)/database?parseTime=true")
	boltFile = flag.String("boltfile", "teian.db", "BoltDB database file to store suggestions")
	loginURL = flag.String("loginurl", "/suggest/login", "Login URL path to redirect to")
	certFile = flag.String("tlscert", "", "TLS public key in PEM format.  Must be used together with -tlskey")
	keyFile  = flag.String("tlskey", "", "TLS private key in PEM format.  Must be used together with -tlscert")
	// Set after flag parsing based on certFile & keyFile.
	useTLS bool
)

const description = `Usage: teian [options]
  A service that allows users to submit suggestions.
Options:
`

func usage() {
	fmt.Fprintf(os.Stderr, description)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
}

func main() {
	flag.Usage = usage
	flag.Parse()
	useTLS = *certFile != "" && *keyFile != ""

	// create database connection and store
	s := datastore.Open(*dbDriver, *dbConfig, *boltFile)
	closeStoreOnSignal(s)
	// add store to context
	ctx := store.NewContext(context.Background(), s)

	http.Handle("/suggest", shimmie.Auth(ctx, serveIndex, *loginURL))
	http.Handle("/suggest/admin", shimmie.Auth(ctx, serveAdmin, *loginURL))
	http.Handle("/suggest/submit", newHandler(ctx, submitHandler))
	http.Handle("/suggest/login", http.HandlerFunc(serveLogin))
	http.Handle("/suggest/login/submit", newHandler(ctx, handleLogin))
	http.Handle("/suggest/logout", http.HandlerFunc(handleLogout))

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

func closeStoreOnSignal(s store.Store) {
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

type ctxHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func newHandler(ctx context.Context, fn ctxHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(ctx, w, r)
	}
}

func serveIndex(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	render(w, suggestionTmpl, nil)
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	render(w, loginTmpl, nil)
}

func serveAdmin(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	u, ok := ctx.Value("user").(*teian.User)
	if !ok {
		http.Redirect(w, r, *loginURL, http.StatusFound)
		return
	}
	if u.Admin != "Y" {
		http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
		return
	}
	suggs, err := store.GetAllSugg(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
	}
	render(w, listTmpl, suggs)
}

func handleLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	user, err := store.GetUser(ctx, username)
	if err != nil {
		log.Println(err)
		render(w, loginTmpl, "User does not exist")
		return
	}
	hash := md5.Sum([]byte(username + password))
	passwordHash := fmt.Sprintf("%x", hash)
	if user.Pass != passwordHash {
		render(w, loginTmpl, "Username and password do not match")
		return
	}
	addr := strings.Split(r.RemoteAddr, ":")[0]
	cookieValue := shimmie.CookieValue(passwordHash, addr)
	shimmie.SetCookie(w, "shm_user", username)
	shimmie.SetCookie(w, "shm_session", cookieValue)
	http.Redirect(w, r, "/suggest", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	shimmie.SetCookie(w, "shm_user", "")
	shimmie.SetCookie(w, "shm_session", "")
	http.Redirect(w, r, "/suggest", http.StatusFound)
}

func submitHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	// redirect if suggestion text is empty
	if len(strings.TrimSpace(text)) == 0 {
		http.Redirect(w, r, "/suggest", http.StatusFound)
		return
	}

	// get user cookie to find out username
	userCookie, err := r.Cookie("shm_user")
	if err != nil || userCookie.Value == "" {
		log.Print("empty or no user cookie found when submiting suggestion")
		http.Redirect(w, r, *loginURL, http.StatusFound)
		return
	}

	type result struct {
		Err error
		Msg string
	}

	// create and store suggestion
	username := userCookie.Value
	err = store.CreateSugg(ctx, username, &teian.Sugg{Text: text})
	if err != nil {
		render(w, submitTmpl, result{Err: err, Msg: "Error: Something broke :'( Our developers were notified. Sorry!"})
	}

	render(w, submitTmpl, result{Msg: "Success: Your suggestion has been submitted. Thank you for your feedback!"})

}

func render(w http.ResponseWriter, t *template.Template, data interface{}) {
	if err := t.Execute(w, data); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var (
	suggestionTmpl = template.Must(template.New("").Parse(baseTemplate + suggestionTemplate + logoutTemplate))
	submitTmpl     = template.Must(template.New("").Parse(baseTemplate + submitTemplate + logoutTemplate))
	listTmpl       = template.Must(template.New("").Parse(baseTemplate + listTemplate + logoutTemplate))
	loginTmpl      = template.Must(template.New("").Parse(baseTemplate + loginTemplate))
)

const (
	baseTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>suggest</title>
	<style>
		label, textarea, input {
			display: block;
		}
	</style>
</head>
<body>
	{{block "content" .}}{{end}}
</body>
</html>
`
	logoutTemplate = `
{{define "logout"}}
<form method="post" action="/suggest/logout">
     <button type="submit">Logout</button>
</form>
{{end}}
`
	suggestionTemplate = `
{{define "content"}}
<form method="post" action="/suggest/submit">
	<label for="text">Write your suggestion</label>
	<textarea id="text" class="large" rows="20" cols="80" name="text" placeholder="Write your suggestion here."></textarea>
	<input type="submit">
</form>
{{block "logout" .}}{{end}}
{{end}}
`
	submitTemplate = `
{{define "content"}}
{{ if .Err }}
<!-- Error -->
<!-- .Err  -->
<!----------->
{{end}}
{{ if .Msg }}
<em>{{.Msg}}</em>
{{end}}
<a href="/suggest">New suggetion</a>
{{block "logout" .}}{{end}}
{{end}}
`
	listTemplate = `
{{define "content"}}
{{ range $k, $v := . }}
	<div class="suggestion">
		<span>{{$v.FmtCreated}} by <a href="https://kusubooru.com/user/{{$v.Username}}">{{$v.Username}}</a></span>
		<textarea cols="80" readonly>{{$v.Text}}</textarea>
	</div>
{{ end }}
{{block "logout" .}}{{end}}
{{end}}
`

	loginTemplate = `
{{define "content"}}
<h1>Login</h1>
<form method="post" action="/suggest/login/submit">
    <label for="username">User name</label>
    <input type="text" id="username" name="username">
    <label for="password">Password</label>
    <input type="password" id="password" name="password">
    <button type="submit">Login</button>
</form>
{{if .}}
<em>{{.}}</em>
{{end}}
{{end}}
`
)
