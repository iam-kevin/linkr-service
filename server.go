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

	shortenerBaseUrl := os.Getenv("LINKR_BASE_URL")
	if shortenerBaseUrl == "" {
		log.Fatal(fmt.Errorf("LINKR_BASE_URL not defined"))
		return
	}

	r.Route("/v1/api", func(r chi.Router) {
		apiHandler := service.NewApiHandler(db, linkr.NewShortner(shortenerBaseUrl))

		// creates a link
		r.Post("/create", apiHandler.HandleCreateLink)

		// creates a client
		r.Post("/client/create", apiHandler.HandleCreateClient)
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}
