// Package api holds HTTP handlers, routing, and middleware for the backend.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// Server holds the dependencies HTTP handlers need.
type Server struct {
	store     *store.Store
	storage   library.Storage
	cfg       library.Config
	logger    *slog.Logger
	jwtSecret string
}

func NewServer(st *store.Store, storage library.Storage, cfg library.Config, logger *slog.Logger, jwtSecret string) *Server {
	return &Server{store: st, storage: storage, cfg: cfg, logger: logger, jwtSecret: jwtSecret}
}

// Router builds the HTTP handler tree. Go 1.22+'s http.ServeMux supports
// method + path-parameter patterns directly, so no router library (e.g.
// chi) is needed for routes this simple.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Public: health and the two auth endpoints (you can't have a token yet).
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)

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

	return s.withLogging(s.withCORS(mux))
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
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
