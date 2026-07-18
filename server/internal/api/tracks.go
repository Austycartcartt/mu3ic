package api

import (
	"net/http"
	"strconv"
)

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.store.ListTracks(r.Context())
	if err != nil {
		s.logger.Error("listing tracks", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list tracks")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

// handleStream serves the raw audio file via http.ServeContent, which
// parses Range headers and returns 206 Partial Content automatically —
// no hand-rolled range parsing needed.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
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

	file, err := s.storage.Open(track.StorageKey)
	if err != nil {
		s.logger.Error("opening track file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to open track")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		s.logger.Error("stat track file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to read track")
		return
	}

	// Content-Type comes from the DB, not from sniffing the filename —
	// storage keys are bare UUIDs with no extension for ServeContent to
	// go on, and the DB value is authoritative anyway (resolved once, at
	// ingest time, from the multipart header + content sniffing).
	w.Header().Set("Content-Type", track.MimeType)
	http.ServeContent(w, r, "", info.ModTime(), file)
}
