package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// pathID parses a {…} path segment as an int64, writing a 400 and
// returning ok=false when it isn't one.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

func (s *Server) handleListPlaylists(w http.ResponseWriter, r *http.Request) {
	playlists, err := s.store.ListPlaylists(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		s.logger.Error("listing playlists", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list playlists")
		return
	}
	writeJSON(w, http.StatusOK, playlists)
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "playlist name is required")
		return
	}

	playlist, err := s.store.CreatePlaylist(r.Context(), userIDFromContext(r.Context()), name)
	if err != nil {
		s.logger.Error("creating playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create playlist")
		return
	}
	writeJSON(w, http.StatusCreated, playlist)
}

func (s *Server) handleRenamePlaylist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "playlist name is required")
		return
	}

	err := s.store.RenamePlaylist(r.Context(), id, userIDFromContext(r.Context()), name)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "playlist not found")
		return
	}
	if err != nil {
		s.logger.Error("renaming playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to rename playlist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	err := s.store.DeletePlaylist(r.Context(), id, userIDFromContext(r.Context()))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "playlist not found")
		return
	}
	if err != nil {
		s.logger.Error("deleting playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete playlist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePlaylistTracks returns the playlist's tracks in order. It calls
// GetPlaylist first so a missing/non-owned playlist is a 404, distinct
// from an existing-but-empty playlist (which is a 200 with []).
func (s *Server) handlePlaylistTracks(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	userID := userIDFromContext(r.Context())

	if _, err := s.store.GetPlaylist(r.Context(), id, userID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "playlist not found")
		return
	} else if err != nil {
		s.logger.Error("getting playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load playlist")
		return
	}

	tracks, err := s.store.ListPlaylistTracks(r.Context(), id, userID)
	if err != nil {
		s.logger.Error("listing playlist tracks", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list playlist tracks")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

func (s *Server) handleAddPlaylistTrack(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		TrackID int64 `json:"track_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.TrackID == 0 {
		writeJSONError(w, http.StatusBadRequest, "track_id is required")
		return
	}

	err := s.store.AddTrackToPlaylist(r.Context(), id, userIDFromContext(r.Context()), body.TrackID)
	if errors.Is(err, sql.ErrNoRows) {
		// Either the playlist or the track isn't the caller's — one 404.
		writeJSONError(w, http.StatusNotFound, "playlist or track not found")
		return
	}
	if err != nil {
		s.logger.Error("adding track to playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add track")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemovePlaylistTrack(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	trackID, ok := pathID(w, r, "trackId")
	if !ok {
		return
	}

	err := s.store.RemoveTrackFromPlaylist(r.Context(), id, userIDFromContext(r.Context()), trackID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "playlist not found")
		return
	}
	if err != nil {
		s.logger.Error("removing track from playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to remove track")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReorderPlaylist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		TrackIDs []int64 `json:"track_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	err := s.store.ReorderPlaylist(r.Context(), id, userIDFromContext(r.Context()), body.TrackIDs)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "playlist not found")
	case errors.Is(err, store.ErrReorderMismatch):
		writeJSONError(w, http.StatusBadRequest, "track_ids must match the playlist's current tracks")
	case err != nil:
		s.logger.Error("reordering playlist", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to reorder playlist")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
