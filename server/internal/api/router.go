// Package api holds HTTP handlers, routing, and middleware for the backend.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// defaultStreamURLTTL is the lifetime of a presigned stream/artwork URL
// when Options.StreamURLTTL is left zero. Short, so a URL that leaks into
// a log or browser history stops working quickly.
const defaultStreamURLTTL = 15 * time.Minute

// RegistrationPolicy controls who may create an account via
// POST /api/auth/register. See handleRegister for how the fields combine.
type RegistrationPolicy struct {
	// Open lets anyone register. Intended for throwaway/staging only.
	Open bool
	// InviteCode, when non-empty, requires the register request to carry a
	// matching "inviteCode" field. Ignored when Open is true.
	InviteCode string
}

// Options is the full set of inputs NewServer needs. It's a struct rather
// than a positional parameter list because Phase 8 (deployment) added
// several knobs — registration policy, proxy trust, presigned-URL TTL.
type Options struct {
	Store        *store.Store
	Storage      library.Storage
	Config       library.Config
	Logger       *slog.Logger
	JWTSecret    string
	Registration RegistrationPolicy
	// TrustProxy makes the server read the client IP from the X-Real-IP
	// header (set it only when a trusted reverse proxy populates that
	// header, as the Phase 8 Caddy config does).
	TrustProxy bool
	// StreamURLTTL is the lifetime of presigned stream/artwork URLs. Zero
	// means defaultStreamURLTTL.
	StreamURLTTL time.Duration
}

// Server holds the dependencies HTTP handlers need.
type Server struct {
	store        *store.Store
	storage      library.Storage
	cfg          library.Config
	logger       *slog.Logger
	jwtSecret    string
	registration RegistrationPolicy
	trustProxy   bool
	streamURLTTL time.Duration
	authLimiter  *ipRateLimiter
}

func NewServer(o Options) *Server {
	ttl := o.StreamURLTTL
	if ttl <= 0 {
		ttl = defaultStreamURLTTL
	}
	return &Server{
		store:        o.Store,
		storage:      o.Storage,
		cfg:          o.Config,
		logger:       o.Logger,
		jwtSecret:    o.JWTSecret,
		registration: o.Registration,
		trustProxy:   o.TrustProxy,
		streamURLTTL: ttl,
		authLimiter:  newIPRateLimiter(),
	}
}

// Router builds the HTTP handler tree. Go 1.22+'s http.ServeMux supports
// method + path-parameter patterns directly, so no router library (e.g.
// chi) is needed for routes this simple.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Public: health and the two auth endpoints (you can't have a token yet).
	// The auth endpoints are rate-limited per client IP so a stolen or
	// leaked deployment URL can't be brute-forced for credentials.
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.Handle("POST /api/auth/register", s.withRateLimit(http.HandlerFunc(s.handleRegister)))
	mux.Handle("POST /api/auth/login", s.withRateLimit(http.HandlerFunc(s.handleLogin)))

	// Everything else requires a valid token. s.protected wraps each
	// handler in withAuth, which puts the caller's user id in the request
	// context for the handler to scope its queries by.
	mux.Handle("POST /api/tracks", s.protected(s.handleUpload))
	mux.Handle("GET /api/tracks", s.protected(s.handleList))
	mux.Handle("GET /api/tracks/{id}/stream", s.protected(s.handleStream))
	mux.Handle("GET /api/tracks/{id}/artwork", s.protected(s.handleArtwork))
	mux.Handle("GET /api/artists", s.protected(s.handleListArtists))
	mux.Handle("GET /api/artists/{name}/tracks", s.protected(s.handleArtistTracks))
	mux.Handle("GET /api/albums", s.protected(s.handleListAlbums))
	mux.Handle("GET /api/albums/{name}/tracks", s.protected(s.handleAlbumTracks))
	mux.Handle("GET /api/search", s.protected(s.handleSearch))
	mux.Handle("GET /api/playlists", s.protected(s.handleListPlaylists))
	mux.Handle("POST /api/playlists", s.protected(s.handleCreatePlaylist))
	mux.Handle("PATCH /api/playlists/{id}", s.protected(s.handleRenamePlaylist))
	mux.Handle("DELETE /api/playlists/{id}", s.protected(s.handleDeletePlaylist))
	mux.Handle("GET /api/playlists/{id}/tracks", s.protected(s.handlePlaylistTracks))
	mux.Handle("POST /api/playlists/{id}/tracks", s.protected(s.handleAddPlaylistTrack))
	mux.Handle("PUT /api/playlists/{id}/tracks", s.protected(s.handleReorderPlaylist))
	mux.Handle("DELETE /api/playlists/{id}/tracks/{trackId}", s.protected(s.handleRemovePlaylistTrack))

	return s.withRequestID(s.withLogging(s.withCORS(mux)))
}

// statusRecorder wraps http.ResponseWriter to remember the status code
// (and whether one was written) so withLogging can log it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		// r.URL.Path (not RawQuery) so a ?token= on the stream/artwork
		// URLs never reaches the logs.
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration", time.Since(start),
			"request_id", requestIDFromContext(r.Context()),
		)
	})
}

// requestIDKey carries the per-request id set by withRequestID.
type requestIDCtxKey struct{}

// withRequestID assigns each request a short random id, echoes it in the
// X-Request-Id response header, and stashes it on the context so
// withLogging can include it. Cheap, and it makes a single log line or a
// user's bug report traceable through the server.
func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf [8]byte
		id := "unknown"
		if _, err := rand.Read(buf[:]); err == nil {
			id = hex.EncodeToString(buf[:])
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDCtxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDCtxKey{}).(string)
	return id
}

// withCORS allows the Expo web client (served from a different origin/port
// than the API) to call this server from the browser. Auth is via a
// bearer token in the Authorization header (or a ?token= query param for
// the stream/artwork URLs the audio player fetches), never cookies, so
// reflecting any Origin without Allow-Credentials is safe here.
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleHealth reports liveness plus a quick database round-trip, so an
// uptime monitor pointed at it catches a server that's up but has lost
// its database.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		s.logger.Warn("health: database ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
