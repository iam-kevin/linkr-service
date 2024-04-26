package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	linkr "iam-kevin/linkr/pkg"
	"iam-kevin/linkr/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	r := chi.NewMux()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	db, err := sqlx.Open("sqlite3", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(fmt.Errorf("unable to create connection to db: %s", err.Error()))
		return
	}

	// TODO: should have this step ran once on server start
	db.MustExec(`INSERT OR IGNORE INTO "Namespace" VALUES (NULL, ?, NULL)`, linkr.ReservedGlobalChar)

	// pull default namespace
	dfNamespace := new(service.LinkrNamespace)
	err = db.Get(dfNamespace, `SELECT * FROM "Namespace" where unique_tag = ?`, linkr.ReservedGlobalChar)
	if err != nil {
		log.Fatalf("couldn't initialize the default namespace: %s", err)
		return
	}

	shortenerBaseUrl := os.Getenv("LINKR_BASE_URL")
	if shortenerBaseUrl == "" {
		log.Fatal(fmt.Errorf("LINKR_BASE_URL not defined"))
		return
	}

	r.Route("/v1/api", func(r chi.Router) {
		apiHandler := service.NewApiHandler(db, linkr.NewShortner(shortenerBaseUrl), dfNamespace)

		// creates a link
		r.Post("/create", apiHandler.HandleCreateLink)

		// creates a client
		r.Post("/client/create", apiHandler.HandleCreateClient)
	})

	r.Route("/", func(r chi.Router) {
		linkHandler := service.NewLinkHandler(db, dfNamespace)

		r.Get("/{id}", linkHandler.HandleRedirectShortenedLink)
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}
