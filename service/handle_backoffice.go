// Responsible for the backoffice things
package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	linkr "iam-kevin/linkr/pkg"

	"github.com/jmoiron/sqlx"
	"github.com/lucsky/cuid"
)

type ApiHandler struct {
	// db connection
	db *sqlx.DB

	// short url host
	shortner *linkr.Shortner
}

func NewApiHandler(db *sqlx.DB, shortner *linkr.Shortner) *ApiHandler {
	return &ApiHandler{
		db:       db,
		shortner: shortner,
	}
}

// helps generate a client who can access and create resources
func generateClient(scope string) linkr.Client {
	scp := linkr.RoleReadOnly
	if scope != "" {
		scp = scope
	}

	// TODO: check scope if known enum

	// generate id
	cid := cuid.New()

	return linkr.Client{
		Id:         fmt.Sprintf("api_%s-%s", cid, time.Now().Format("YYYYMMDD")),
		Scope:      scp,
		SigningKey: base64.URLEncoding.EncodeToString(sha256.New().Sum(nil)),
	}
}

// Handler for creating a resource user
func (a *ApiHandler) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	body := new(RequestClientCreate)
	json.NewDecoder(r.Body).Decode(body)

	insertClientStr := `INSERT INTO "ApiClient" (id, scope, sigining_key, created_at, updated_at) VALUES (?, ?, ?, now(), now())`

	c := generateClient(body.ClientType)
	a.db.MustExec(insertClientStr, c.Id, c.Scope, c.SigningKey)

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(ResponseClientCreate{
		Message: "client created created",
		Details: c,
	})
}

// Handler for creating shortned links
func (a *ApiHandler) HandleCreateLink(w http.ResponseWriter, r *http.Request) {
	input := new(RequestLinkCreate)
	json.NewDecoder(r.Body).Decode(input)

	var namespaceId int64

	if input.Namespace != "" {
		// check if namespace is valid
		if input.Namespace == linkr.ReservedGlobalChar {
			http.Error(w, "not supported", http.StatusBadRequest)
			return
		}

		// TODO: other checks
		// - \w
		// - not long (4 chars max)

		res := a.db.MustExec(`INSERT INTO "Namespace" (unique_tag) VALUES (?)`, input.Namespace)
		ix, _ := res.LastInsertId()
		namespaceId = ix
	} else {
		// TODO: should have this step ran once on server start
		res := a.db.MustExec(`INSERT OR IGNORE INTO "Namespace" (unique_tag) VALUES (?)`, linkr.ReservedGlobalChar)
		ix, _ := res.LastInsertId()
		namespaceId = ix
	}

	var expiresIn int64 = 0
	now := time.Now()
	var expiresAt time = nil

	if input.ExpiresIn != "" {
		ex, err := linkr.ConvertStringDurationToSeconds(input.ExpiresIn)
		if err != nil {
			http.Error(w, fmt.Sprintf("couldn't construction duration from `expires_in` input: %s", err.Error()), http.StatusBadRequest)
			return
		}

		expiresIn = int64(ex.Seconds())
		expiresAt = now.Add(ex)
	}

	// create url
	urlshort := cuid.Slug()
	serializedHeaders := extractHeadersToForward(r.Header.Clone())

	// save the link
	a.db.MustExec(`
		INSERT INTO "Link" 
			(identifier, destination_url, namespace_id, expires_in, expires_at, headers) 
			VALUES
			(?, ?, ?, ?, ?)
	`, urlshort, input.Url, namespaceId, expiresIn, expiresAt, serializedHeaders)

	shortenedLink := ""
	if input.Namespace == "" {
		shortenedLink = a.shortner.Create(urlshort)
	} else {
		shortenedLink = a.shortner.CreateWithNamespace(input.Namespace, urlshort)
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(ResponseClientCreate{
		Message: "link created",
		Details: ResponseLinkCreate{
			ShortenedUrl:     shortenedLink,
			Identifier:       urlshort,
			Namespace:        input.Namespace,
			ExpiresInSeconds: int(expiresIn),
			CreatedAt:        now.Format(time.RFC3339),
			ExpiresAt:        expiresAt.Format(time.RFC3339),
		},
	})
}

const (
	// Prefix to be atteched to request headers that
	// we'd like to forward as part of the request
	//
	// Example if the saved header is `Linkr-Forward-Super-Secret: 2313`,
	// the forwarded request becomes `Super-Secret: 2313`
	LinkrHeaderPrefix = "Linkr-Forward"
)

type forwardedHeaders struct {
	headerString string
}

// extract the headers to forward
func extractHeadersToForward(headers http.Header) *string {
	linkrPrefix := strings.ToLower(LinkrHeaderPrefix)
	newheaders := make(http.Header)
	for hk, hv := range headers {
		lowerhk := strings.ToLower(hk)
		if suffix, ok := strings.CutPrefix(lowerhk, fmt.Sprintf("%s-", linkrPrefix)); ok {
			// remove trailing -
			newheaders.Set(suffix, strings.Join(hv, ","))
		}
	}

	if len(newheaders) == 0 {
		return nil
	}

	// simple serialization
	// to store result as k1=v11,v12;k2=v21,v22
	serialized := make([]string, len(newheaders))
	for k, v := range newheaders {
		// comma separate values and
		// potential store as k=v1,v2,v3
		serialized = append(serialized, fmt.Sprintf("%s=%s", k, strings.Join(v, ",")))
	}

	output := (strings.Join(serialized, ";"))
	return &output
}
