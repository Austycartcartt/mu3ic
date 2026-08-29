// Package auth handles password hashing and stateless JWT session tokens.
// It has no HTTP types, so it's callable directly from tests and from the
// api package's handlers/middleware alike.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// tokenTTL is how long an issued token stays valid. 24h for now (MVP);
// after that the client re-logs in. See docs/PHASE-5-authentication.md.
const tokenTTL = 24 * time.Hour

// ErrInvalidToken is returned by ParseToken for any token that fails to
// parse, has a bad signature, uses an unexpected signing method, or is
// expired — the caller doesn't need to tell those apart (all mean "log in
// again").
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims is the JWT payload: just the user id plus the registered
// (standard) claims for expiry/issued-at.
type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// GenerateToken issues an HS256 token for userID, signed with secret,
// expiring tokenTTL from now.
func GenerateToken(userID int64, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

// TokenTTL exposes the token lifetime so handlers can report expiresAt to
// the client without hard-coding the duration in two places.
func TokenTTL() time.Duration { return tokenTTL }

// ParseToken verifies tokenString against secret and returns the user id
// it carries. Any failure (bad signature, wrong algorithm, expired,
// malformed) comes back as ErrInvalidToken.
func ParseToken(tokenString, secret string) (int64, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		// Reject anything that isn't HMAC — without this check a token
		// with alg:"none", or an RS256 token whose "key" is our public
		// secret string, could be accepted.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}
	if claims.UserID == 0 {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}

// HashPassword returns a bcrypt hash of plain, suitable for storing in
// users.password_hash.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether plain matches the stored bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
