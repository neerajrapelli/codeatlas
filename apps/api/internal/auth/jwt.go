package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingSecret = errors.New("JWT_SECRET is required when auth is enabled")
	ErrInvalidToken  = errors.New("invalid or expired token")
)

// Claims are validated on each API request.
type Claims struct {
	Sub      string `json:"sub"`
	TenantID string `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

// Validator verifies HS256 JWTs issued by your identity provider or CodeAtlas bootstrap endpoint.
type Validator struct {
	secret []byte
}

func NewValidator(secret string) (*Validator, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrMissingSecret
	}
	return &Validator{secret: []byte(secret)}, nil
}

func (v *Validator) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// IssueToken creates a signed JWT for bootstrap/dev (production should use your IdP).
func IssueToken(secret, subject, tenantID string, ttl time.Duration) (string, error) {
	v, err := NewValidator(secret)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := Claims{
		Sub:      subject,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(v.secret)
}
