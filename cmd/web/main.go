package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"forum/config"
	"forum/internal/models"
	"forum/internal/store"

	_ "github.com/mattn/go-sqlite3"
)

type application struct {
	errorLog         *log.Logger
	infoLog          *log.Logger
	posts            *models.PostModel
	templateCache    map[string]*template.Template
	users            *models.UserModel
	sessionManager   *SessionManager
	postTags         *models.CategoryModel
	comments         *models.CommentModel
	reactions        *models.ReactionModel
	commentReactions *models.ReactionCommentModel
	imageUpload      *models.ImageModel
	TokenBucket      *TokenBucket
}

func main() {
	config, err := config.NewConfig()
	if err != nil {
		return
	}
	db, err := store.NewSqlite3(config)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	defer db.Close()
	f, err := os.OpenFile("./info.log", os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	infoLog := log.New(f, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}
	app := &application{
		errorLog:         errorLog,
		infoLog:          infoLog,
		posts:            &models.PostModel{DB: db},
		users:            &models.UserModel{DB: db},
		templateCache:    templateCache,
		sessionManager:   &SessionManager{DB: db},
		postTags:         &models.CategoryModel{DB: db},
		comments:         &models.CommentModel{DB: db},
		reactions:        &models.ReactionModel{DB: db},
		commentReactions: &models.ReactionCommentModel{DB: db},
		imageUpload:      &models.ImageModel{DB: db},
	}
	if err != nil {
		panic(err)
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		MaxVersion:       tls.VersionTLS12,
	}

	server := &http.Server{
		Addr:           ":8081",
		ErrorLog:       errorLog,
		Handler:        app.routes(),
		MaxHeaderBytes: 524288,
		TLSConfig:      tlsConfig,
		IdleTimeout:    time.Minute,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
	}
	infoLog.Printf("Starting server on %s", server.Addr)
	err = server.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}
