package api

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
)

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.store.ListTracks(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		s.logger.Error("listing tracks", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list tracks")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

// handleStream serves a track's audio. When the storage backend can
// presign (R2), it 302-redirects the client to a short-lived direct URL
// so the bytes never transit this server. Otherwise (filesystem) it
// serves them via http.ServeContent, which parses Range headers and
// returns 206 Partial Content automatically.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	// GetTrack is scoped to the caller, so someone else's track id looks
	// exactly like a missing one — both land here as a 404.
	track, err := s.store.GetTrack(r.Context(), id, userIDFromContext(r.Context()))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "track not found")
		return
	}

	s.serveObject(w, r, track.StorageKey, track.MimeType, track.UploadedAt)
}

// serveObject streams (or redirects to) a stored object. contentType is
// authoritative — it's set from the DB, since storage keys are bare
// UUIDs with no extension to sniff and the DB value was resolved once at
// ingest time. modTime is the DB timestamp (stable across backends) used
// as the cache validator on the filesystem path.
func (s *Server) serveObject(w http.ResponseWriter, r *http.Request, key, contentType string, modTime time.Time) {
	if p, ok := s.storage.(library.Presigner); ok {
		url, err := p.PresignGet(r.Context(), key, contentType, s.streamURLTTL)
		if err != nil {
			s.logger.Error("presigning object url", "error", err, "key", key)
			writeJSONError(w, http.StatusInternalServerError, "failed to open track")
			return
		}
		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	obj, err := s.storage.Open(r.Context(), key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "track not found")
			return
		}
		s.logger.Error("opening object", "error", err, "key", key)
		writeJSONError(w, http.StatusInternalServerError, "failed to open track")
		return
	}
	defer obj.Close()

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "", modTime, obj)
}
