// Responsible for the backoffice things
package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
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

	dfNs *LinkrNamespace
}

func NewApiHandler(db *sqlx.DB, shortner *linkr.Shortner, defaultNs *LinkrNamespace) *ApiHandler {
	return &ApiHandler{
		db:       db,
		shortner: shortner,
		dfNs:     defaultNs,
	}
}

const (
	// set time format
	TimeFormatYYYYMMDD = "20060102"
)

// helps generate a client who can access and create resources
func generateClient(scope string) (*linkr.Client, error) {
	scp := linkr.RoleReadOnly
	if scope != "" {
		scp = scope
	}

	if !linkr.IsRole(scope) {
		return nil, fmt.Errorf("unknown role type '%s'", scope)
	}

	// generate id
	cid := cuid.New()

	return &linkr.Client{
		Id:         fmt.Sprintf("api_%s-%s", cid, time.Now().Format(TimeFormatYYYYMMDD)),
		Scope:      scp,
		SigningKey: base64.URLEncoding.EncodeToString(sha256.New().Sum(nil)),
	}, nil
}

// Handler for creating a resource user
func (a *ApiHandler) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	body := new(RequestClientCreate)
	json.NewDecoder(r.Body).Decode(body)

	// default role
	roleType := linkr.RoleWriteOnly
	if linkr.IsRole(body.Role) {
		roleType = body.Role
	}

	insertClientStr := `INSERT INTO "ApiClient" (id, username, description, scope, signing_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	c, err := generateClient(roleType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a.db.MustExec(insertClientStr, c.Id, body.Username, nil, c.Scope, c.SigningKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(ResponseClientCreate{
		Message: "client created",
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

		// check namespace exissts
		ns := new(LinkrNamespace)
		err := a.db.Get(ns, `SELECT * FROM "Namespace" where unique_tag = ?`, input.Namespace)
		if err == nil {
			// assumes exists
			namespaceId = ns.Id
		} else {
			res := a.db.MustExec(`INSERT OR IGNORE INTO "Namespace" (unique_tag) VALUES (?)`, input.Namespace)
			ix, _ := res.LastInsertId()
			namespaceId = ix
		}

	} else {
		namespaceId = a.dfNs.Id
	}

	slog.Info("namespace id", "namespaceid", namespaceId)

	var expiresIn int64 = 0
	now := time.Now()
	var expiresAt *time.Time = nil

	if input.ExpiresIn != "" {
		ex, err := linkr.ConvertStringDurationToSeconds(input.ExpiresIn)
		if err != nil {
			http.Error(w, fmt.Sprintf("couldn't construction duration from `expires_in` input: %s", err.Error()), http.StatusBadRequest)
			return
		}

		expiresIn = int64(ex.Seconds())
		v := now.Add(ex)
		expiresAt = &v
	}

	// create url
	urlshort := cuid.Slug()
	serializedHeaders := extractHeadersToForward(r.Header.Clone())

	// save the link
	a.db.MustExec(`
		INSERT INTO "Link" 
			(id, identifier, destination_url, namespace_id, expires_in, expires_at, headers) 
			VALUES
			(NULL, ?, ?, ?, ?, ?, ?)
	`, urlshort, input.Url, namespaceId, expiresIn, &expiresAt, serializedHeaders)

	shortenedLink := ""
	if input.Namespace == "" {
		shortenedLink = a.shortner.Create(urlshort)
	} else {
		shortenedLink = a.shortner.CreateWithNamespace(input.Namespace, urlshort)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)

	var expiresInSecond *int64
	var expiresAtString string
	if expiresAt != nil {
		expiresAtString = (*expiresAt).Format(time.RFC3339)
	}

	if expiresIn > 0 {
		expiresInSecond = &expiresIn
	}

	json.NewEncoder(w).Encode(ResponseClientCreate{
		Message: "link created",
		Details: ResponseLinkCreate{
			ShortenedUrl:     shortenedLink,
			Identifier:       urlshort,
			Namespace:        input.Namespace,
			ExpiresInSeconds: expiresInSecond,
			CreatedAt:        now.Format(time.RFC3339),
			ExpiresAt:        expiresAtString,
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
