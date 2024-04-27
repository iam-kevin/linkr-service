package service

import (
	"encoding/base64"
	"log/slog"

	"github.com/gbrlsnchs/jwt/v3"
)

type LinkrJwtVerifier struct {
	m *jwt.HMACSHA
}

func NewVerifier(b64SignignKey string) (*LinkrJwtVerifier, error) {
	signignKey, err := base64.URLEncoding.DecodeString(b64SignignKey)
	slog.Info("signing key", "key", signignKey)
	if err != nil {
		return nil, err
	}

	return &LinkrJwtVerifier{
		m: jwt.NewHS256(signignKey),
	}, nil
}

type LinkrJWToken struct {
	Payload jwt.Payload
	Body    string `json:"body"`
}

// verifies the payload with the expected payload
func (j *LinkrJwtVerifier) Verify(digest []byte, body string, subject string) error {
	ltoken := new(LinkrJWToken)
	ltoken.Body = body
	ltoken.Payload.Subject = subject

	slog.Info("checking payload", "digest", string(digest), "subject", subject)

	_, err := jwt.Verify(
		digest,
		j.m,
		ltoken,
		jwt.ValidatePayload(
			&ltoken.Payload,
			jwt.SubjectValidator(subject),
		),
	)

	return err
}
