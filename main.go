package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	httpAddr = flag.String("http", "localhost:8080", "HTTP listen address")
	dbDriver = flag.String("driver", "mysql", "Database driver")
	dbConfig = flag.String("config", "", "username:password@(host:port)/database?parseTime=true")
)

type Suggestion struct {
	ID       int64
	Username string
	Text     string
	Created  time.Time
}

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

	// create database connection and store
	s := datastore.New(*dbDriver, *dbConfig)
	// add store to context
	ctx := store.NewContext(context.Background(), s)

	http.Handle("/suggest/submit", http.HandlerFunc(submitHandler))
	if err := http.ListenAndServe(*httpAddr, nil); err != nil {
		log.Fatalf("Could not start listening on %v: %v", *httpAddr, err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	render(w, suggestionTmpl, nil)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	text := r.PostFormValue("text")
	if len(strings.TrimSpace(text)) == 0 {
		http.Redirect(w, r, "/suggest", http.StatusFound)
		return
	}
	// store suggestion here
	render(w, submitTmpl, nil)
}

func render(w http.ResponseWriter, t *template.Template, data interface{}) {
	if err := t.Execute(w, data); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var (
	baseTmpl       = template.Must(template.New("baseTmpl").Parse(baseTemplate))
	suggestionTmpl = template.Must(template.Must(baseTmpl.Clone()).Parse(suggestionTemplate))
	submitTmpl     = template.Must(template.Must(baseTmpl.Clone()).Parse(submitTemplate))
	listTmpl       = template.Must(template.Must(baseTmpl.Clone()).Parse(listTemplate))

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
	suggestionTemplate = `
{{define "content"}}
<form method="post" action="/suggest/submit">
	<label for="text">Enter your suggestion</label>
	<textarea id="text" class="large" rows="20" cols="80" name="text"></textarea>
	<input type="submit">
</form>
{{end}}
`
	submitTemplate = `
{{define "content"}}
Your suggestion has been submitted. Thank you for your feedback.
{{end}}
`
	listTemplate = `
{{define "content"}}
list placeholder	
{{end}}
`
)
