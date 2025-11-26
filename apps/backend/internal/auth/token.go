package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// SurrealClaims captures the SurrealDB-specific JWT fields alongside standard registered claims.
type SurrealClaims struct {
	ID string `json:"ID"`
	NS string `json:"NS"`
	DB string `json:"DB"`
	AC string `json:"AC"`
	jwt.RegisteredClaims
}

// ParseToken validates the Surreal-issued JWT and returns its strongly-typed claims.
func ParseToken(tokenString, secret string) (*SurrealClaims, error) {
	if tokenString == "" {
		return nil, errors.New("missing token")
	}
	if secret == "" {
		return nil, errors.New("jwt secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &SurrealClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS512.Alg() {
			return nil, fmt.Errorf("unexpected signing method %s", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*SurrealClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
