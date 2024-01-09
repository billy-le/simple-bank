package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrorInvalidToken = errors.New("token is invalid")
	ErrorExpiredToken = errors.New("token is expired")
)

type Payload struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewPayload(username string, duration time.Duration) (*Payload, error) {
	tokenId, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	claims := jwt.RegisteredClaims{
		ID:        tokenId.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}

	return &Payload{
		Username:         username,
		RegisteredClaims: claims,
	}, nil
}

// Valid checks if the token payload is valid or not
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiresAt.Time) {
		return ErrorExpiredToken
	}
	return nil
}
