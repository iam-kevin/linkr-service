// Handles the logic and processes to do with the
// shortened URL
package service

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type LinkrNamespace struct {
	Id          int            `db:"id"`
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
	id                int           `db:"id"`
	Tag               string        `db:"identifier"`
	OriginalUrl       string        `db:"destination_url"`
	ExpiresAt         sql.NullInt32 `db:"expires_at"`
	ExpiresIn         sql.NullInt32 `db:"expires_in"`
	CreatedAt         time.Time     `db:"created_at"`
	SerializedHeaders string        `db:"headers"`
}

// redirect to the page
// shortned id in {id}
func (l *LinkHandler) HandleRedirectShortenedLink(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// check if such a thing exists
	link := new(Link)
	err := l.db.Get(link, `SELECT * FROM "Link" WHERE identifier = ? AND namespace_id = ?`, id, l.dfNs.Id)
	if err != nil {
		http.Error(w, "no such thing", http.StatusNotFound)
		return
	}

	// TODO: deserialize the header

	req, err := http.NewRequest(http.MethodGet, link.OriginalUrl, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't forward the payload : %s", err.Error()), http.StatusNotFound)
		return

	}

	http.Redirect(w, req, link.OriginalUrl, http.StatusTemporaryRedirect)
}
