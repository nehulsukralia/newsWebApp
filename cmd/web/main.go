package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	_ "github.com/lib/pq"
	"github.com/nehulsukralia/newsWebApp/models"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/postgresql"
)

const (
	sessionKeyUserId   = "userId"
	sessionKeyUserName = "userName"
)

type application struct {
	appName string
	server  server
	debug   bool
	infoLog *log.Logger
	errLog  *log.Logger
	view    *jet.Set
	session *scs.SessionManager
	Models  models.Models
}

type server struct {
	host string
	port string
	url  string
}

func main() {
	migrate := flag.Bool("migrate", false, "should migrate - drop all tables")

	flag.Parse()

	server := server{
		host: "localhost",
		port: "8009",
		url:  "http://localhost:8009",
	}

	dbase, err := openDB("postgres://root:root@localhost/hnews?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer dbase.Close()

	// init upper/db
	upper, err := postgresql.New(dbase)
	if err != nil {
		log.Fatal(err)
	}
	defer func(upper db.Session) {
		err := upper.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(upper)

	// run migration after db setup
	if *migrate {
		fmt.Println("Running migration")
		err = runMigrate(upper)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Done running migration")
	}

	app := &application{
		appName: "News",
		server:  server,
		debug:   true,
		infoLog: log.New(os.Stdout, "INFO\t", log.Ltime|log.Ldate|log.Lshortfile),
		errLog:  log.New(os.Stderr, "ERROR\t", log.Ltime|log.Ldate|log.Llongfile),
		Models:  models.New(upper),
	}

	// init jet template
	if app.debug {
		app.view = jet.NewSet(jet.NewOSFileSystemLoader("./views"), jet.InDevelopmentMode()) //to avoid to re run the server after every change made while debugging
	} else {
		app.view = jet.NewSet(jet.NewOSFileSystemLoader("./views"))
	}

	//init session
	app.session = scs.New()

	app.session.Lifetime = 24 * time.Hour
	app.session.Cookie.Name = app.appName
	app.session.Cookie.Domain = app.server.host
	app.session.Cookie.Persist = true
	app.session.Cookie.SameSite = http.SameSiteStrictMode
	app.session.Store = postgresstore.New(dbase) // to store session data in postgres db

	if err := app.listenAndServer(); err != nil {
		log.Fatal(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func runMigrate(db db.Session) error {
	script, err := os.ReadFile("./migrations/tables.sql")
	if err != nil {
		return err
	}

	_, err = db.SQL().Exec(string(script))

	return err
}
