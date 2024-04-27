package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	linkr "iam-kevin/linkr/pkg"

	"github.com/jmoiron/sqlx"
)

type CommandCenter struct {
	db *sqlx.DB
}

func NewCommandCenter(db *sqlx.DB) *CommandCenter {
	return &CommandCenter{
		db: db,
	}
}

// set context key type
type contextKey string

const (
	// context value holding reference to Linkr Client
	CtxLinkrClient contextKey = "CTX_LINKR_CLIENT"
)

// Middleware that checks if the user if authenticated
func (cc *CommandCenter) MiddlewareGated(next http.Handler) http.Handler {
	// ...
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ..
		apiKey := r.Header.Get(HeaderLinkrApiKey)
		if apiKey == "" {
			http.Error(w, "missing api key", http.StatusForbidden)
			return
		}
		digestString := r.Header.Get(HeaderLinkrDigest)
		if digestString == "" {
			http.Error(w, "missing request digest", http.StatusBadRequest)
			return
		}

		clientKeyByte, err := base64.StdEncoding.DecodeString(string(apiKey))
		if err != nil {
			slog.Error(fmt.Sprintf("failed to base64 parse the key, reason: %s", err.Error()))
			http.Error(w, "invalid authentication", http.StatusForbidden)
			return
		}

		// check the authentication
		client := new(LinkrClient)
		err = cc.db.Get(client, `SELECT * FROM "ApiClient" where id = ?`, string(clientKeyByte))
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "invalid authentication", http.StatusForbidden)
			return
		}

		v, err := NewVerifier(client.SigningKey)
		if err != nil {
			slog.Error(fmt.Sprintf("couldn't initialize verifier: %s", err.Error()))
			http.Error(w, "something went wrong. please try again later", http.StatusInternalServerError)
			return
		}

		// clone request body
		var buf bytes.Buffer
		io.Copy(&buf, r.Body)
		payload, err := io.ReadAll(&buf)
		if err != nil {
			slog.Error(fmt.Sprintf("failed verify payload: %s", err.Error()))
			http.Error(w, "invalid authentication", http.StatusForbidden)
			return
		}

		digest, err := base64.URLEncoding.DecodeString(digestString)
		if err != nil {
			slog.Error(fmt.Sprintf("failed verify payload: %s", err.Error()))
			http.Error(w, "invalid authentication", http.StatusForbidden)
			return
		}

		// check digest
		err = v.Verify(digest, string(payload), string(clientKeyByte))
		if err != nil {
			slog.Error(fmt.Sprintf("coudn't verify payload. reason: %s", err.Error()))
			http.Error(w, "failed to verify payload", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), CtxLinkrClient, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// [Must be used under `MiddlewareGated`]
// Check if the user is of roles `roleTypes`
func (cc *CommandCenter) MiddlewareWithRoles(roleTypes ...string) func(http.Handler) http.Handler {
	for _, roleType := range roleTypes {
		if !linkr.IsRole(roleType) {
			panic(fmt.Sprintf("no such role type '%s'. only support %v", roleType, linkr.SupportedListOfRoles()))
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			client := r.Context().Value(CtxLinkrClient).(*LinkrClient)

			if client == nil {
				slog.Error("user entity is not attached as part of the request")
				http.Error(w, "something went wrong. please try again later", http.StatusInternalServerError)
				return
			}

			if !includes(roleTypes, client.Role) {
				http.Error(w, "operation not allowed", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func includes[T comparable](haystack []T, needle T) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}

type LinkrClient struct {
	Id         string    `db:"id"`
	Role       string    `db:"scope"`
	SigningKey string    `db:"signing_key"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
