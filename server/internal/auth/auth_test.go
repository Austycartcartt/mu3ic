package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseToken(t *testing.T) {
	const secret = "test-secret"

	token, err := GenerateToken(42, secret)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	uid, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if uid != 42 {
		t.Errorf("uid = %d, want 42", uid)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken(1, "right-secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := ParseToken(token, "wrong-secret"); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseToken_Expired(t *testing.T) {
	const secret = "test-secret"
	// Build a token that expired an hour ago directly, since GenerateToken
	// always sets a future expiry.
	claims := Claims{
		UserID: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing expired token: %v", err)
	}
	if _, err := ParseToken(token, secret); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseToken_NonHMACRejected(t *testing.T) {
	const secret = "test-secret"
	// A token with alg "none" must not be accepted even though the claims
	// look fine.
	claims := Claims{UserID: 5}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none token: %v", err)
	}
	if _, err := ParseToken(token, secret); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseToken_Garbage(t *testing.T) {
	if _, err := ParseToken("not.a.jwt", "test-secret"); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("hash equals plaintext")
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Error("CheckPassword() = false for the correct password")
	}
	if CheckPassword(hash, "wrong password") {
		t.Error("CheckPassword() = true for a wrong password")
	}
}
