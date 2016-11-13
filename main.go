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
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/shimmie/store"
	"github.com/kusubooru/teian/teian"
	"github.com/kusubooru/teian/teian/boltstore"
)

const theVersion = "1.0.0"

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
		},
	}

	http.Handle("/suggest", shim.Auth(app.serveIndex, *loginURL))
	http.Handle("/suggest/admin", shim.Auth(app.serveAdmin, *loginURL))
	http.Handle("/suggest/admin/delete", shim.Auth(app.handleDelete, *loginURL))
	http.Handle("/suggest/submit", shim.Auth(app.handleSubmit, *loginURL))
	http.Handle("/suggest/login", http.HandlerFunc(app.serveLogin))
	http.Handle("/suggest/login/submit", http.HandlerFunc(app.handleLogin))
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
	app.render(w, listTmpl, suggs)
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
		http.Redirect(w, r, *loginURL, http.StatusFound)
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

var (
	suggestionTmpl = template.Must(template.New("suggestionTmpl").Parse(baseTemplate + subnavTemplate + suggestionTemplate))
	submitTmpl     = template.Must(template.New("submitTmpl").Parse(baseTemplate + subnavTemplate + submitTemplate))
	listTmpl       = template.Must(template.New("listTmpl").Parse(baseTemplate + subnavTemplate + toolbarTemplate + listTemplate))
	loginTmpl      = template.Must(template.New("loginTmpl").Parse(baseTemplate + loginTemplate))
)

const (
	baseTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>suggest</title>
	<script type='text/javascript'>
		var _gaq = _gaq || [];
		_gaq.push(['_setAccount', '{{.Conf.AnalyticsID}}']);
		_gaq.push(['_trackPageview']);
		(function() {
		  var ga = document.createElement('script'); ga.type = 'text/javascript'; ga.async = true;
		  ga.src = ('https:' == document.location.protocol ? 'https://ssl' : 'http://www') + '.google-analytics.com/ga.js';
		  var s = document.getElementsByTagName('script')[0]; s.parentNode.insertBefore(ga, s);
		})();
	</script>
	<meta name="description" content="{{.Conf.Description}}">
	<meta name="keywords" content="{{.Conf.Keywords}}">
	<style>
		* {
			font-size: 16px; 
			line-height: 1.2;
			font-family: Verdana, Geneva, sans-serif;
		}
		a:link {
			color:#006FFA;
			text-decoration:none;
		}
		a:visited {
			color:#006FFA;
			text-decoration:none;
		}
		a:hover {
			color:#33CFFF;
			text-decoration:none;
		}
		a:active {
			color:#006FFA;
			text-decoration:none;
		}

		p, a, span {font-size: 120%;}

		body {
			list-style-type: none;
			padding-top: 0;
			margin-top: 0;
		}

		#site-title {
			font-size: 133%;
			padding: 0.5em;
			margin: 0;
		}

		#subnav {
		    background: #f6f6f6;
			padding-top: 1em;
			padding-bottom: 1em;
			border-top: 1px #ebebeb solid;
			border-bottom: 1px #ebebeb solid;
		}

		#subnav a {
		    padding: 0.5em;
		}

		#subnav subnav-button-link {
		    padding: 0.5em;
		}

		.subnav-button-form {
			display: inline;
		}

		.subnav-button-link {
			background: none!important;
			border: none;
			padding: 0!important;
			font: inherit;
			font-size: 120%;
			cursor: pointer;
			color: #006FFA;
			display: inline;
		}

		.subnav-button-link:visited {
			color:#006FFA;
			text-decoration:none;
		}
		.subnav-button-link:hover {
			color:#33CFFF;
			text-decoration:none;
		}
		.subnav-button-link:active {
			color:#006FFA;
			text-decoration:none;
		}

		#login-form label, #login-form input, #login-form button, #login-form em {
			padding: 0.5em;
			display: block;
			font-size: 120%;
			line-height:1.2;
		}

		#login-form button {
			margin-top: 0.5em;
		}

		#login-form h1 {
			padding: 0.5em;
			font-size: 120%;
		}

		.toolbar {
			padding: 0.5em;
		}
		.toolbar input, .toolbar button {
			font-size: 120%;
		}

		.suggestion {
			padding: 0.5em;
			border-top: 1px #ebebeb solid;
			border-bottom: 1px #ebebeb solid;
			border-left: 0.3em #006FFA solid;
			border-top-left-radius: 0.3em;
			border-bottom-left-radius: 0.3em;
			line-height: 200%;
		}

		.suggestion form {
			display: inline;
		}

		.suggestion textarea {
			display: block;
			font-size: 120%;
			line-height:1.2;
		}
		.suggestion:nth-of-type(even) {
		    background: #f6f6f6;
		}

		.suggestion-form {
			padding: 0.5em;
			line-height: 200%;
		}

		textarea {
			width: 70%;
		}
		@media (max-width: 768px) {
			textarea {
				width: 100%;
			}
		}

		.suggestion-form input[type=submit] {
			padding: 0.5em;
			margin-top: 0.5em;
			display: block;
		}

		.alert {
			border-radius: 4px;
			padding: 1em;
			margin-top: 0.5em;
			margin-bottom: 0.5em;
			font-size: 120%;
			width: 70%;
		}
		.alert strong {
			font-size: inherit;
		}
		@media (max-width: 768px) {
			.alert{
				width: 90%;
				padding-left:5%;
				padding-right:5%;
			}
		}

		.alert-success {
			color: #3c763d;
			background-color: #dff0d8;
			border-color: #d6e9c6;
		}

		.alert-error {
			color: #a94442;
			background-color: #f2dede;
			border-color: #ebccd1;
		}

		footer {
			color: #ccc;
			font-size: 0.9em;
			padding-left: 0.5em;
			padding-top: 1em;
		}

		footer a {
			font-size: 0.9em;
		}

	</style>
</head>
<body>
	<h1 id="site-title"><a href="/post/list">{{.Conf.SiteTitle}}</a></h1>
	{{block "subnav" .}}{{end}}
	{{block "toolbar" .}}{{end}}
	{{block "content" .}}{{end}}
	<footer>
		{{block "footer" .}}{{end}}
		<em>Served by <a href="https://github.com/kusubooru/teian">teian</a> v{{.Conf.Version}}</em>
	</footer>
</body>
</html>
`
	toolbarTemplate = `
{{define "toolbar"}}
<div class="toolbar">
	<form method="get" action="/suggest/admin">
		<input type="text" name="u" placeholder="Username">
		<input type="text" name="t" placeholder="Text">
		<label for="order">Order By</label>
		<select id="order" name="o">
			<option value="dd">Date Desc</option>
			<option value="da">Date Asc</option>
			<option value="ud">Username Desc</option>
			<option value="ua">Username Asc</option>
		</select>
	    <button type="submit">Search</button>
		<input type="reset" value="Reset">
	</form>
</div>
{{end}}
`
	subnavTemplate = `
{{define "subnav"}}
<div id="subnav">
	<a href="/suggest">New suggestion</a>
	<form class="subnav-button-form" method="post" action="/suggest/logout">
	     <input class="subnav-button-link" type="submit" value="Logout">
	</form>
</div>
{{end}}
`
	suggestionTemplate = `
{{define "content"}}
<div class="suggestion-form">
	<form method="post" action="/suggest/submit">
		<p>{{.Conf.WriteMsg}}</p>
		<textarea class="large" rows="20" cols="80" name="text" placeholder="Write your suggestion here."></textarea>
		<input type="submit">
	</form>
</div>
{{end}}
`
	submitTemplate = `
{{define "content"}}
{{ if .Data.Err }}
<!-- Error -->
<!-- .Data.Err  -->
<!----------->
{{end}}
{{ if .Data.Msg }}
<div class="alert alert-{{.Data.Type}}">
	<strong>
	{{if eq .Data.Type "success"}}
	Success:
	{{else}}
	Error:
	{{end}}
	</strong>{{.Data.Msg}}
</div>
{{end}}
{{end}}
`
	listTemplate = `
{{define "content"}}
{{ range $k, $v := .Data }}
	<div class="suggestion">
		<span>{{$v.FmtCreated}} by <a href="/user/{{$v.Username}}">{{$v.Username}}</a></span>
		<form method="post" action="/suggest/admin/delete">
			<input type="hidden" name="username" value="{{$v.Username}}">
			<input type="hidden" name="id" value="{{$v.ID}}">
			<input type="submit" value="Delete">
		</form>
		<textarea cols="80" readonly>{{$v.Text}}</textarea>
	</div>
{{ end }}
{{end}}
`

	loginTemplate = `
{{define "content"}}
<form id="login-form" method="post" action="/suggest/login/submit">
<h1>Login</h1>
    <label for="username">User name</label>
    <input type="text" id="username" name="username">
    <label for="password">Password</label>
    <input type="password" id="password" name="password">
    <button type="submit">Login</button>
	{{if .Data}}
	<em>{{.Data}}</em>
	{{end}}
</form>
{{end}}
`
)
