package api

import "net/http"

func (s *Server) handleListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := s.store.ListAlbums(r.Context())
	if err != nil {
		s.logger.Error("listing albums", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list albums")
		return
	}
	writeJSON(w, http.StatusOK, albums)
}

// handleAlbumTracks returns the tracks for the given album name, optionally
// disambiguated by an ?artist= query param (matched against each track's
// album_artist, not its own artist) since album titles aren't unique across
// artists. An unknown album just yields an empty list, not a 404 — same
// reasoning as handleArtistTracks.
func (s *Server) handleAlbumTracks(w http.ResponseWriter, r *http.Request) {
	album := r.PathValue("name")
	artist := r.URL.Query().Get("artist")
	tracks, err := s.store.ListTracksByAlbum(r.Context(), album, artist)
	if err != nil {
		s.logger.Error("listing tracks by album", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list album tracks")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}
