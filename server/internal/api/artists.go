package api

import "net/http"

func (s *Server) handleListArtists(w http.ResponseWriter, r *http.Request) {
	artists, err := s.store.ListArtists(r.Context())
	if err != nil {
		s.logger.Error("listing artists", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list artists")
		return
	}
	writeJSON(w, http.StatusOK, artists)
}

// handleArtistTracks returns the tracks for the given artist name. An
// unknown artist just yields an empty list, not a 404 — artists aren't a
// normalized entity with existence independent of tracks, unlike a track id.
func (s *Server) handleArtistTracks(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tracks, err := s.store.ListTracksByArtist(r.Context(), name)
	if err != nil {
		s.logger.Error("listing tracks by artist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list artist tracks")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}
