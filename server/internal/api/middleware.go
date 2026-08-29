package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/Austycartcartt/mu3ic/server/internal/auth"
)

// ctxKey is an unexported type for context keys defined in this package,
// so they can't collide with keys from any other package.
type ctxKey int

const userIDKey ctxKey = iota

// protected wraps a handler func in withAuth. Written as an explicit
// adapter (rather than a cleverer generic helper) so the route table in
// Router() still reads as one line per route.
func (s *Server) protected(h http.HandlerFunc) http.Handler {
	return s.withAuth(h)
}

// withAuth rejects any request without a valid token. The token is taken
// from the Authorization: Bearer header when present, otherwise from a
// ?token= query param — the latter is there for the /stream and /artwork
// URLs, which the audio player and <Image> fetch directly with no way to
// set a header. On success the caller's user id is stored in the request
// context for the wrapped handler to read via userIDFromContext.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := auth.ParseToken(token, s.jwtSecret)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header value, or returns "" if the header is missing or not a bearer.
func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}

// userIDFromContext returns the authenticated user's id. It is only valid
// on a request that passed through withAuth (i.e. any handler registered
// via s.protected); calling it elsewhere returns 0.
func userIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}
