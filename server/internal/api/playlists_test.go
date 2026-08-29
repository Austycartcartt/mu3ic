package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// createPlaylist drives handleCreatePlaylist and returns the new row.
func createPlaylist(t *testing.T, s *Server, userID int64, name string) store.Playlist {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":`+jsonString(name)+`}`))
	rec := httptest.NewRecorder()
	s.handleCreatePlaylist(rec, reqAs(userID, req))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create playlist status = %d, want 201 (body %s)", rec.Code, rec.Body)
	}
	var p store.Playlist
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decoding created playlist: %v", err)
	}
	// No explicit cleanup: testServer's per-test user is deleted on
	// cleanup, and playlists.user_id cascades.
	return p
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func addTrack(t *testing.T, s *Server, userID, playlistID, trackID int64) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"track_id":%d}`, trackID)
	req := httptest.NewRequest(http.MethodPost, "/api/playlists/x/tracks", strings.NewReader(body))
	req.SetPathValue("id", fmt.Sprint(playlistID))
	rec := httptest.NewRecorder()
	s.handleAddPlaylistTrack(rec, reqAs(userID, req))
	return rec
}

func playlistTrackIDs(t *testing.T, s *Server, userID, playlistID int64) []int64 {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/playlists/x/tracks", nil)
	req.SetPathValue("id", fmt.Sprint(playlistID))
	rec := httptest.NewRecorder()
	s.handlePlaylistTracks(rec, reqAs(userID, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("list playlist tracks status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	var tracks []store.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatalf("decoding playlist tracks: %v", err)
	}
	ids := make([]int64, len(tracks))
	for i, tr := range tracks {
		ids[i] = tr.ID
	}
	return ids
}

func TestPlaylistCreateAndList(t *testing.T) {
	s, _, _, user := testServer(t)

	p := createPlaylist(t, s, user.ID, "Roadtrip")
	if p.Name != "Roadtrip" || p.TrackCount != 0 {
		t.Fatalf("created playlist = %+v, want name Roadtrip / 0 tracks", p)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
	rec := httptest.NewRecorder()
	s.handleListPlaylists(rec, reqAs(user.ID, req))

	var playlists []store.Playlist
	if err := json.Unmarshal(rec.Body.Bytes(), &playlists); err != nil {
		t.Fatalf("decoding playlists: %v", err)
	}
	var found bool
	for _, pl := range playlists {
		if pl.ID == p.ID {
			found = true
			if pl.TrackCount != 0 {
				t.Errorf("TrackCount = %d, want 0", pl.TrackCount)
			}
		}
	}
	if !found {
		t.Fatalf("created playlist %d not in list %+v", p.ID, playlists)
	}
}

func TestPlaylistAddTracksInOrder(t *testing.T) {
	s, db, _, user := testServer(t)
	p := createPlaylist(t, s, user.ID, "Mix")
	t1 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "One"})
	t2 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Two"})
	t3 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Three"})

	for _, tr := range []store.Track{t1, t2, t3} {
		if rec := addTrack(t, s, user.ID, p.ID, tr.ID); rec.Code != http.StatusNoContent {
			t.Fatalf("add track %d status = %d, want 204 (body %s)", tr.ID, rec.Code, rec.Body)
		}
	}

	got := playlistTrackIDs(t, s, user.ID, p.ID)
	want := []int64{t1.ID, t2.ID, t3.ID}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("playlist order = %v, want insertion order %v", got, want)
	}
}

func TestPlaylistAddDuplicateIsNoop(t *testing.T) {
	s, db, _, user := testServer(t)
	p := createPlaylist(t, s, user.ID, "Dupes")
	tr := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Only"})

	for i := 0; i < 2; i++ {
		if rec := addTrack(t, s, user.ID, p.ID, tr.ID); rec.Code != http.StatusNoContent {
			t.Fatalf("add #%d status = %d, want 204", i, rec.Code)
		}
	}
	if got := playlistTrackIDs(t, s, user.ID, p.ID); len(got) != 1 {
		t.Fatalf("playlist has %d tracks after double add, want 1", len(got))
	}
}

func TestPlaylistAddForeignTrackIs404(t *testing.T) {
	s, db, _, owner := testServer(t)
	other := createTestUser(t, s, db, uniqueEmail(t))
	p := createPlaylist(t, s, owner.ID, "Mine")
	foreign := insertTestTrack(t, s, db, other.ID, store.NewTrack{Title: "Not yours"})

	if rec := addTrack(t, s, owner.ID, p.ID, foreign.ID); rec.Code != http.StatusNotFound {
		t.Fatalf("adding another user's track: status = %d, want 404 (body %s)", rec.Code, rec.Body)
	}
}

func TestPlaylistReorder(t *testing.T) {
	s, db, _, user := testServer(t)
	p := createPlaylist(t, s, user.ID, "Reorder me")
	t1 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "A"})
	t2 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "B"})
	t3 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "C"})
	for _, tr := range []store.Track{t1, t2, t3} {
		addTrack(t, s, user.ID, p.ID, tr.ID)
	}

	reorder := func(ids []int64) int {
		b, _ := json.Marshal(map[string][]int64{"track_ids": ids})
		req := httptest.NewRequest(http.MethodPut, "/api/playlists/x/tracks", strings.NewReader(string(b)))
		req.SetPathValue("id", fmt.Sprint(p.ID))
		rec := httptest.NewRecorder()
		s.handleReorderPlaylist(rec, reqAs(user.ID, req))
		return rec.Code
	}

	if code := reorder([]int64{t3.ID, t1.ID, t2.ID}); code != http.StatusNoContent {
		t.Fatalf("reorder status = %d, want 204", code)
	}
	got := playlistTrackIDs(t, s, user.ID, p.ID)
	if fmt.Sprint(got) != fmt.Sprint([]int64{t3.ID, t1.ID, t2.ID}) {
		t.Fatalf("order after reorder = %v, want [%d %d %d]", got, t3.ID, t1.ID, t2.ID)
	}

	// A list that isn't the exact membership is rejected.
	if code := reorder([]int64{t1.ID, t2.ID}); code != http.StatusBadRequest {
		t.Errorf("short reorder list status = %d, want 400", code)
	}
	if code := reorder([]int64{t1.ID, t2.ID, t2.ID}); code != http.StatusBadRequest {
		t.Errorf("duplicate reorder list status = %d, want 400", code)
	}
}

func TestPlaylistRemoveTrack(t *testing.T) {
	s, db, _, user := testServer(t)
	p := createPlaylist(t, s, user.ID, "Trim")
	t1 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Keep"})
	t2 := insertTestTrack(t, s, db, user.ID, store.NewTrack{Title: "Drop"})
	addTrack(t, s, user.ID, p.ID, t1.ID)
	addTrack(t, s, user.ID, p.ID, t2.ID)

	req := httptest.NewRequest(http.MethodDelete, "/api/playlists/x/tracks/y", nil)
	req.SetPathValue("id", fmt.Sprint(p.ID))
	req.SetPathValue("trackId", fmt.Sprint(t2.ID))
	rec := httptest.NewRecorder()
	s.handleRemovePlaylistTrack(rec, reqAs(user.ID, req))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d, want 204 (body %s)", rec.Code, rec.Body)
	}

	got := playlistTrackIDs(t, s, user.ID, p.ID)
	if fmt.Sprint(got) != fmt.Sprint([]int64{t1.ID}) {
		t.Fatalf("tracks after remove = %v, want [%d]", got, t1.ID)
	}
}

func TestPlaylistRenameAndDelete(t *testing.T) {
	s, _, _, user := testServer(t)
	p := createPlaylist(t, s, user.ID, "Old name")

	req := httptest.NewRequest(http.MethodPatch, "/api/playlists/x", strings.NewReader(`{"name":"New name"}`))
	req.SetPathValue("id", fmt.Sprint(p.ID))
	rec := httptest.NewRecorder()
	s.handleRenamePlaylist(rec, reqAs(user.ID, req))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("rename status = %d, want 204", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/playlists/x", nil)
	req.SetPathValue("id", fmt.Sprint(p.ID))
	rec = httptest.NewRecorder()
	s.handleDeletePlaylist(rec, reqAs(user.ID, req))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}

	// Gone now.
	req = httptest.NewRequest(http.MethodGet, "/api/playlists/x/tracks", nil)
	req.SetPathValue("id", fmt.Sprint(p.ID))
	rec = httptest.NewRecorder()
	s.handlePlaylistTracks(rec, reqAs(user.ID, req))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted playlist tracks status = %d, want 404", rec.Code)
	}
}

func TestPlaylistForeignAccessIs404(t *testing.T) {
	s, db, _, owner := testServer(t)
	other := createTestUser(t, s, db, uniqueEmail(t))
	p := createPlaylist(t, s, owner.ID, "Owner's")

	cases := []struct {
		name string
		call func() int
	}{
		{"rename", func() int {
			req := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(`{"name":"hax"}`))
			req.SetPathValue("id", fmt.Sprint(p.ID))
			rec := httptest.NewRecorder()
			s.handleRenamePlaylist(rec, reqAs(other.ID, req))
			return rec.Code
		}},
		{"delete", func() int {
			req := httptest.NewRequest(http.MethodDelete, "/x", nil)
			req.SetPathValue("id", fmt.Sprint(p.ID))
			rec := httptest.NewRecorder()
			s.handleDeletePlaylist(rec, reqAs(other.ID, req))
			return rec.Code
		}},
		{"list tracks", func() int {
			req := httptest.NewRequest(http.MethodGet, "/x/tracks", nil)
			req.SetPathValue("id", fmt.Sprint(p.ID))
			rec := httptest.NewRecorder()
			s.handlePlaylistTracks(rec, reqAs(other.ID, req))
			return rec.Code
		}},
	}
	for _, c := range cases {
		if code := c.call(); code != http.StatusNotFound {
			t.Errorf("%s by non-owner: status = %d, want 404", c.name, code)
		}
	}
}
