package api

import (
	"net/http"
	"strconv"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
)

// handleArtwork serves a track's embedded cover art. Like handleStream it
// 302-redirects to a presigned URL when the storage backend supports it,
// otherwise streams the bytes via http.ServeContent. The Content-Type is
// set from the stored extension (only ".jpg"/".png" ever occur) rather
// than sniffed, since object keys carry no extension.
func (s *Server) handleArtwork(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	track, err := s.store.GetTrack(r.Context(), id, userIDFromContext(r.Context()))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "track not found")
		return
	}
	if !track.HasArtwork {
		writeJSONError(w, http.StatusNotFound, "track has no artwork")
		return
	}

	s.serveObject(w, r, track.StorageKey+track.ArtworkExt, library.ArtworkContentType(track.ArtworkExt), track.UploadedAt)
}
