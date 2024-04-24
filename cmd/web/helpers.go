package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type ErrorStatus struct {
	Status  int
	Message string
}

type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, err
		}
	}

	return f, nil
}

func (app *application) authenticatedUser(r *http.Request) int {
	sessionCookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			return 0
		}

		if err == sessionCookie.Valid() {
			return 0
		}
	}

	return app.sessionManager.GetUserIDBySessionToken(sessionCookie.Value)
}

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n", err.Error())
	app.errorLog.Output(2, trace)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	tmpl, err := template.ParseFiles("./ui/html/pages/error.tmpl")
	if err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
	w.WriteHeader(status)
	ErrorMessage := ErrorStatus{status, http.StatusText(status)}
	err = tmpl.Execute(w, ErrorMessage)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(status), status)
		return
	}
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) render(w http.ResponseWriter, status int, page string, data *templateData) {
	// Retrieve the appropriate template set from the cache based on the page
	// name (like 'home.tmpl'). If no entry exists in the cache with the
	// provided name, then create a new error and call the serverError() helper
	// method that we made earlier and return.
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}
	// Write out the provided HTTP status code ('200 OK', '400 Bad Request'
	// etc).
	buf := new(bytes.Buffer)
	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, err)
		return
	}
	w.WriteHeader(status)
	// Execute the template set and write the response body. Again, if there
	// is any error we call the the serverError() helper.
	buf.WriteTo(w)
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		CurrentYear: time.Now().Year(),
	}
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {
	if td == nil {
		td = &templateData{}
	}
	td.AuthenticatedUser = app.authenticatedUser(r)
	td.CurrentYear = time.Now().Year()
	return td
}

type TokenBucket struct {
	rate       float64
	capacity   float64
	tokens     float64
	lastAccess time.Time
	mu         sync.Mutex
}

func NewTokenBucket(rate, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:     rate,
		capacity: capacity,
		tokens:   capacity,
	}
}

func (tb *TokenBucket) Take(tokens float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	tb.tokens += tb.rate * now.Sub(tb.lastAccess).Seconds()
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	if tokens <= tb.tokens {
		tb.tokens -= tokens
		tb.lastAccess = now
		return true
	}

	return false
}

var (
	ClientTokenBucketMap      = make(map[string]*TokenBucket)
	ClientTokenBucketMapMutex sync.Mutex
)

func GetClientTokenBucket(clientAddr string, rate, capacity float64) *TokenBucket {
	ClientTokenBucketMapMutex.Lock()
	defer ClientTokenBucketMapMutex.Unlock()

	if tb, ok := ClientTokenBucketMap[clientAddr]; ok {
		return tb
	}

	// If the client does not have a token bucket, create one and store it in the map.
	tb := NewTokenBucket(rate, capacity)
	ClientTokenBucketMap[clientAddr] = tb
	return tb
}
