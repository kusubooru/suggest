package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/teian/teian"
)

const (
	defaultMaxMemory   = 32 << 20 // 32 MB
	maxFileSize        = 50 << 20 // 50 MB
	uploadFormFileName = "uploadfile"
)

func (app *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Limit max upload.
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Authenticate user.
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	user, err := app.Shimmie.GetUserByName(username)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Wrong username or password.", http.StatusUnauthorized)
			return
		}
		app.Errorf(w, http.StatusInternalServerError, err, "get user %q failed: %v", username, err)
		return
	}
	passwordHash := shimmie.PasswordHash(username, password)
	if user.Pass != passwordHash {
		http.Error(w, "Wrong username or password.", http.StatusUnauthorized)
		return
	}

	file, handler, err := r.FormFile(uploadFormFileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	uploadLocation := filepath.Join(*uploadDir, username)
	if err = os.MkdirAll(uploadLocation, 0700); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := time.Now().Format("2006-01-02_15.04.05.000_") + filepath.Base(handler.Filename)
	f, err := os.Create(filepath.Join(uploadLocation, filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	n, err := io.Copy(f, file)
	if err != nil {
		app.Errorf(w, http.StatusInternalServerError, err, "file copy failed")
		return
	}

	// Check how many bytes the user is allowed to upload.
	remain, err := app.Suggestions.CheckQuota(username, teian.Quota(n))
	if err != nil {
		if rerr := os.Remove(f.Name()); rerr != nil {
			app.Log.Println("file cleanup failed:", rerr)
		}
		if err == teian.ErrOverQuota {
			http.Error(w, "quota exceeded", http.StatusForbidden)
			return
		}
		http.Error(w, "check quota failed", http.StatusInternalServerError)
		return
	}

	go sendMail(username, f.Name(), n, int64(remain), app.Log)
	fmt.Fprintf(w, "%v", remain)
}

func sendMail(username, file string, uploaded, remain int64, logger Logger) {
	path, err := filepath.Abs(file)
	if err != nil {
		path = file
	}
	mail := teian.Mail{
		To:      "kusubooru@gmail.com",
		From:    "teian-tagaa-uploads@kusubooru.com",
		Subject: "Tagaa upload by " + username,
		Content: fmt.Sprintf("User %s uploaded %vMB (%vMB remain)\nat %s",
			username, uploaded/1024/1024, remain/1024/1024, path),
	}
	err = teian.SendMail(mail)
	if err != nil {
		logger.Println(err)
	}
}
