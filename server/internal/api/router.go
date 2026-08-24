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
	store   *store.Store
	storage library.Storage
	cfg     library.Config
	logger  *slog.Logger
}

func NewServer(st *store.Store, storage library.Storage, cfg library.Config, logger *slog.Logger) *Server {
	return &Server{store: st, storage: storage, cfg: cfg, logger: logger}
}

// Router builds the HTTP handler tree. Go 1.22+'s http.ServeMux supports
// method + path-parameter patterns directly, so no router library (e.g.
// chi) is needed for routes this simple.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/tracks", s.handleUpload)
	mux.HandleFunc("GET /api/tracks", s.handleList)
	mux.HandleFunc("GET /api/tracks/{id}/stream", s.handleStream)
	mux.HandleFunc("GET /api/tracks/{id}/artwork", s.handleArtwork)
	mux.HandleFunc("GET /api/artists", s.handleListArtists)
	mux.HandleFunc("GET /api/artists/{name}/tracks", s.handleArtistTracks)
	mux.HandleFunc("GET /api/albums", s.handleListAlbums)
	mux.HandleFunc("GET /api/albums/{name}/tracks", s.handleAlbumTracks)
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
// than the API) to call this server from the browser. There's no
// auth/cookies involved, so reflecting any Origin is safe here.
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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
