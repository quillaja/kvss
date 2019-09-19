// kvss is a simple key-value storage service

package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbpath := flag.String("db", "", "path to sqlite3 database (required)")
	flag.Parse()

	if *dbpath == "" {
		flag.Usage()
		os.Exit(1)
	}

	NewApplication(*dbpath).run(HTTP)
}

type Application struct {
	Routes *http.ServeMux
	DB     *sqlx.DB
	Log    *log.Logger
}

func NewApplication(dbPath string) *Application {
	db, err := sqlx.Open("sqlite3", dbPath)
	kill(err)

	app := &Application{DB: db}
	app.setupRoutes()

	app.Log = log.New(os.Stderr, "", log.LstdFlags)

	return app
}

func (app *Application) run(mode string) {
	defer app.DB.Close()

	switch mode {
	case FCGI:
		err := fcgi.Serve(nil, app.Routes)
		kill(err)

	case HTTP:
		err := http.ListenAndServe(":8000", app.Routes)
		kill(err)
	}
}

func kill(err error) {
	if err != nil {
		panic(err)
	}
}

const (
	FCGI = "fcgi"
	HTTP = "http"
)
