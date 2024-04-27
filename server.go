package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

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

	// NOTE: might want to move this aside
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

	commander := service.NewCommandCenter(db)

	r.Route("/v1/api", func(r chi.Router) {
		apiHandler := service.NewApiHandler(db, linkr.NewShortner(shortenerBaseUrl), dfNamespace)

		// TODO: add ratelimiting in this route group

		// set role within this group
		// admin can do anything. (NOTE: might want to think about this)
		r.Use(commander.MiddlewareGated)

		r.Group(func(r chi.Router) {
			// in this group, set permission for those who
			// can create links
			r.Use(commander.MiddlewareWithRoles(linkr.RoleReadWrite, linkr.RoleWriteOnly, linkr.RoleAdmin))

			// creates a link
			r.Post("/create", apiHandler.HandleCreateLink)
		})

		r.Group(func(r chi.Router) {
			// creates a client
			r.Use(commander.MiddlewareWithRoles(linkr.RoleAdmin))
			r.Post("/client/create", apiHandler.HandleCreateClient)
		})
	})

	r.Route("/", func(r chi.Router) {
		linkHandler := service.NewLinkHandler(db, dfNamespace)

		r.Get("/{namespace}/{id}", linkHandler.HandleRedirectShortenedLinkWithNamespace)
		r.Get("/{id}", linkHandler.HandleRedirectShortenedLink)
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// server endpoint
	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        r,
		BaseContext: func(l net.Listener) context.Context {
			fmt.Printf("[%s] Application running => %s\n", time.Now().Local().String(), l.Addr().String())
			return context.Background()
		},
	}

	log.Fatal(server.ListenAndServe())
}
