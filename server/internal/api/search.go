package api

import (
	"net/http"
	"strings"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// handleSearch does a substring match over the caller's tracks (title,
// artist, album). A blank q returns an empty list rather than every track
// — the client hits this on every keystroke, so "typed nothing" shouldn't
// dump the whole library.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, []store.Track{})
		return
	}

	tracks, err := s.store.SearchTracks(r.Context(), userIDFromContext(r.Context()), query)
	if err != nil {
		s.logger.Error("searching tracks", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}
