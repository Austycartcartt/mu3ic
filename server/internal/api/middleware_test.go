package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Austycartcartt/mu3ic/server/internal/auth"
)

// echoUserID is a trivial protected handler: it writes back the user id
// withAuth put in the context, so a test can confirm the id made it
// through.
func echoUserID(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int64{"uid": userIDFromContext(r.Context())})
}

func TestWithAuth(t *testing.T) {
	const secret = "test-secret"
	s := &Server{jwtSecret: secret}
	handler := s.withAuth(http.HandlerFunc(echoUserID))

	goodToken, err := auth.GenerateToken(99, secret)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	t.Run("valid bearer header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tracks", nil)
		req.Header.Set("Authorization", "Bearer "+goodToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if body := rec.Body.String(); body != `{"uid":99}`+"\n" {
			t.Errorf("body = %q, want the echoed uid", body)
		}
	})

	t.Run("valid token query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tracks/1/stream?token="+goodToken, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("no token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tracks", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("garbage token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/tracks", nil)
		req.Header.Set("Authorization", "Bearer not-a-jwt")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		claims := auth.Claims{
			UserID: 1,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		}
		expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("signing expired token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/tracks", nil)
		req.Header.Set("Authorization", "Bearer "+expired)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("token signed with a different secret", func(t *testing.T) {
		wrong, err := auth.GenerateToken(1, "some-other-secret")
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/tracks", nil)
		req.Header.Set("Authorization", "Bearer "+wrong)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
