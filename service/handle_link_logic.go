// Handles the logic and processes to do with the
// shortened URL
package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	linkr "iam-kevin/linkr/pkg"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type LinkrNamespace struct {
	Id          int64          `db:"id"`
	Tag         string         `db:"unique_tag"`
	Description sql.NullString `db:"desc"`
}

type LinkHandler struct {
	db   *sqlx.DB
	dfNs *LinkrNamespace
}

func NewLinkHandler(db *sqlx.DB, defaultNs *LinkrNamespace) *LinkHandler {
	return &LinkHandler{
		db:   db,
		dfNs: defaultNs,
	}
}

type Link struct {
	Id                int            `db:"id"`
	Tag               string         `db:"identifier"`
	OriginalUrl       string         `db:"destination_url"`
	NamespaceId       int            `db:"namespace_id"`
	ExpiresAt         sql.NullTime   `db:"expires_at"`
	ExpiresIn         sql.NullInt32  `db:"expires_in"`
	CreatedAt         time.Time      `db:"created_at"`
	SerializedHeaders sql.NullString `db:"headers"`
}

// redirect to the page
// shortned id in {id}
func (l *LinkHandler) HandleRedirectShortenedLink(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// check if such a thing exists
	link := new(Link)
	err := l.db.Get(link, `SELECT * FROM "Link" WHERE identifier = ? AND namespace_id = ?`, id, l.dfNs.Id)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't retrieve that: %s", err.Error()), http.StatusNotFound)
		return
	}

	// TODO: deserialize the header

	req, err := http.NewRequest(http.MethodGet, link.OriginalUrl, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't create the request : %s", err.Error()), http.StatusNotFound)
		return

	}

	http.Redirect(w, req, link.OriginalUrl, http.StatusTemporaryRedirect)
}

// redirect to the page
// shortned id in {id}
func (l *LinkHandler) HandleRedirectShortenedLinkWithNamespace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	namespace := chi.URLParam(r, "namespace")

	if namespace == linkr.ReservedGlobalChar {
		http.Error(w, "invalid or unsupported namespace", http.StatusBadRequest)
		return
	}

	// get namespace
	ns := new(LinkrNamespace)
	err := l.db.Get(ns, `SELECT * FROM "Namespace" where unique_tag = ?`, namespace)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "url not found", http.StatusNotFound)
		return
	}

	// check if such a thing exists
	link := new(Link)
	err = l.db.Get(link, `SELECT * FROM "Link" WHERE identifier = ? AND namespace_id = ?`, id, ns.Id)
	if err != nil {
		slog.Error(fmt.Sprintf("couldn't retrieve that: %s", err.Error()))
		http.Error(w, "url not found", http.StatusNotFound)
		return
	}

	// TODO: deserialize the header

	req, err := http.NewRequest(http.MethodGet, link.OriginalUrl, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("couldn't create the request : %s", err.Error()))
		http.Error(w, "url not found", http.StatusNotFound)
		return

	}

	http.Redirect(w, req, link.OriginalUrl, http.StatusTemporaryRedirect)
}
