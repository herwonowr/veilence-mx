// Package token provides JWT token generation and validation.
package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

const (
	// tokenTypeAccess identifies an access token.
	tokenTypeAccess = "access"
)

// claims represents the JWT claims used for authentication tokens.
type claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTProvider implements usecase.TokenProvider using JWT.
type JWTProvider struct {
	secret          []byte
	previousSecrets [][]byte
}

// New creates a new JWTProvider.
// The primary secret is used for signing. previousSecrets are optional older
// secrets accepted for validation during secret rotation.
func New(secret string, previousSecrets ...string) *JWTProvider {
	var prev [][]byte
	for _, s := range previousSecrets {
		if s != "" {
			prev = append(prev, []byte(s))
		}
	}
	return &JWTProvider{
		secret:          []byte(secret),
		previousSecrets: prev,
	}
}

// GenerateAccessToken creates a signed JWT access token.
func (p *JWTProvider) GenerateAccessToken(userID, email string, duration time.Duration) (string, error) {
	c := &claims{
		UserID:    userID,
		Email:     email,
		TokenType: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "veilence-mx",
			Subject:   userID,
			Audience:  jwt.ClaimStrings{"veilence-mx-api"},
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	tokenString, err := t.SignedString(p.secret)
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return tokenString, nil
}

// GenerateRefreshToken creates a cryptographically random refresh token string.
func (p *JWTProvider) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ValidateAccessToken parses and validates a JWT access token, returning its claims.
// It first tries the primary secret, then falls back to previous secrets to
// support seamless JWT secret rotation.
func (p *JWTProvider) ValidateAccessToken(tokenString string) (*usecase.TokenClaims, error) {
	// Try primary secret first
	c, err := validateWithSecret(tokenString, p.secret)
	if err == nil {
		return c, nil
	}

	// Try previous secrets (rotation support)
	for _, prev := range p.previousSecrets {
		c, prevErr := validateWithSecret(tokenString, prev)
		if prevErr == nil {
			slog.Debug("token validated with previous secret (rotation in progress)")
			return c, nil
		}
	}

	return nil, err
}

func validateWithSecret(tokenString string, secret []byte) (*usecase.TokenClaims, error) {
	c := &claims{}
	t, err := jwt.ParseWithClaims(tokenString, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	if !t.Valid {
		return nil, errors.New("invalid token")
	}

	if c.TokenType != tokenTypeAccess {
		return nil, errors.New("not an access token")
	}

	return &usecase.TokenClaims{
		UserID:    c.UserID,
		Email:     c.Email,
		TokenType: c.TokenType,
	}, nil
}
