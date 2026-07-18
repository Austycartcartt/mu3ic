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
	return s.withLogging(mux)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
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
