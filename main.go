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
	dbpath := flag.String("db", "kvss.db", "path to sqlite3 database")
	mode := flag.String("mode", FCGI, "mode to run in (http or fcgi)")
	flag.Parse()

	if *dbpath == "" {
		flag.Usage()
		os.Exit(1)
	}

	NewApplication(*dbpath).run(*mode)
}

// Application holds the important state for the app.
type Application struct {
	Routes  *http.ServeMux
	DB      *sqlx.DB
	Log     *log.Logger
	logfile *os.File
}

// NewApplication creates a new application using the sqlite3 db.
func NewApplication(dbPath string) *Application {
	app := &Application{}

	file, err := os.OpenFile("error.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	kill(err)
	app.Log = log.New(file, "", log.LstdFlags)
	app.logfile = file

	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		app.Log.Println(err)
		os.Exit(1)
	}
	app.DB = db

	app.setupRoutes()

	return app
}

// run starts the server.
func (app *Application) run(mode string) {
	defer app.DB.Close()
	defer app.logfile.Close()

	switch mode {
	case FCGI:
		err := fcgi.Serve(nil, app.Routes)
		if err != nil {
			app.Log.Println(err)
			os.Exit(1)
		}

	case HTTP:
		err := http.ListenAndServe(":8000", app.Routes)
		if err != nil {
			app.Log.Println(err)
			os.Exit(1)
		}
	}
}

func kill(err error) {
	if err != nil {
		panic(err)
	}
}

// used for application run "mode"
const (
	FCGI = "fcgi"
	HTTP = "http"
)
