package api

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
)

// handleArtwork serves a track's embedded cover art via http.ServeFile,
// which sets the correct Content-Type (from the file extension) and
// caching headers automatically — no need to hand-roll either.
func (s *Server) handleArtwork(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	track, err := s.store.GetTrack(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "track not found")
		return
	}
	if !track.HasArtwork {
		writeJSONError(w, http.StatusNotFound, "track has no artwork")
		return
	}

	path, err := s.storage.ArtworkPath(track.StorageKey, track.ArtworkExt)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "track has no artwork")
			return
		}
		s.logger.Error("resolving artwork path", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to open artwork")
		return
	}

	http.ServeFile(w, r, path)
}
