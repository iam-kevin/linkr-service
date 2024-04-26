package linkr

import (
	"fmt"
	"strings"
)

const (
	RoleAdmin     = "admin"
	RoleReadWrite = "read-write"
	RoleReadOnly  = "read-only"
	RoleWriteOnly = "write-only"
)

type Client struct {
	Id         string `json:"client_id"`
	SigningKey string `json:"client_signing_key"`
	Scope      string
}

const Version = 1

const (
	// denotes that the url is not a part of
	// a subset
	ReservedGlobalChar = "-"
)

type Shortner struct {
	baseUrl string
}

func NewShortner(baseUrl string) *Shortner {
	return &Shortner{
		baseUrl: baseUrl,
	}
}

// Creates a shortned link from an identifier
func (s *Shortner) Create(identifier string) string {
	return fmt.Sprintf("%s/%s",
		// remove trailing /
		strings.TrimRightFunc(s.baseUrl, func(r rune) bool {
			return r == '/'
		}),
		strings.TrimFunc(identifier, func(r rune) bool { return r == '/' }))
}

func (s *Shortner) CreateWithNamespace(namespace string, identifier string) string {
	return fmt.Sprintf("%s/%s/%s",
		// remove trailing /
		strings.TrimRightFunc(s.baseUrl, func(r rune) bool {
			return r == '/'
		}),
		strings.TrimFunc(namespace, func(r rune) bool { return r == '/' }),
		strings.TrimFunc(identifier, func(r rune) bool { return r == '/' }))
}
