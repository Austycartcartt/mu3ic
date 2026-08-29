package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

func TestHandleListAlbums_SameTitleDifferentArtist(t *testing.T) {
	s, db, _, user := testServer(t)

	// Two different artists each with an album literally titled "Greatest
	// Hits" must produce two distinct rows, not one merged row — proves
	// ListAlbums groups by (album, artist), not album alone.
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist A", Album: "Greatest Hits"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist B", Album: "Greatest Hits"})

	req := httptest.NewRequest(http.MethodGet, "/api/albums", nil)
	rec := httptest.NewRecorder()
	s.handleListAlbums(rec, reqAs(user.ID, req))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var albums []store.AlbumSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &albums); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	var matching []store.AlbumSummary
	for _, a := range albums {
		if a.Album == "Greatest Hits" {
			matching = append(matching, a)
		}
	}
	if len(matching) != 2 {
		t.Fatalf("len(matching Greatest Hits rows) = %d, want 2", len(matching))
	}
}

func TestHandleListAlbums_VariousArtistsCompilation(t *testing.T) {
	s, db, _, user := testServer(t)

	// A various-artists compilation: same album, same explicit AlbumArtist,
	// but a different per-track Artist on every row (e.g. from filename
	// parsing during a bulk folder upload). This must collapse into a
	// single album row, not one row per distinct track artist — that was
	// the bug: grouping by (album, artist) instead of (album, album_artist).
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Aeikus", Album: "Going Deeper, Vol. 7", AlbumArtist: "Various Artists", Title: "Cataracta"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Other DJ", Album: "Going Deeper, Vol. 7", AlbumArtist: "Various Artists", Title: "Another Track"})

	req := httptest.NewRequest(http.MethodGet, "/api/albums", nil)
	rec := httptest.NewRecorder()
	s.handleListAlbums(rec, reqAs(user.ID, req))

	var albums []store.AlbumSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &albums); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	var matching []store.AlbumSummary
	for _, a := range albums {
		if a.Album == "Going Deeper, Vol. 7" {
			matching = append(matching, a)
		}
	}
	if len(matching) != 1 {
		t.Fatalf("len(matching Going Deeper rows) = %d, want 1, got %+v", len(matching), matching)
	}
	if matching[0].Artist != "Various Artists" {
		t.Errorf("Artist = %q, want %q", matching[0].Artist, "Various Artists")
	}
	if matching[0].TrackCount != 2 {
		t.Errorf("TrackCount = %d, want 2", matching[0].TrackCount)
	}

	// The album's track listing must still return both differently-artist
	// tracks, since it's filtered by album_artist, not each track's artist.
	req = httptest.NewRequest(http.MethodGet, "/api/albums/x/tracks?artist=Various+Artists", nil)
	req.SetPathValue("name", "Going Deeper, Vol. 7")
	rec = httptest.NewRecorder()
	s.handleAlbumTracks(rec, reqAs(user.ID, req))

	var tracks []store.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("len(tracks) = %d, want 2", len(tracks))
	}
}

func TestHandleAlbumTracks_ArtistDisambiguation(t *testing.T) {
	s, db, _, user := testServer(t)

	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist A", Album: "Greatest Hits", Title: "A Song"})
	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist B", Album: "Greatest Hits", Title: "B Song"})

	// With ?artist= given, only that artist's tracks come back.
	req := httptest.NewRequest(http.MethodGet, "/api/albums/x/tracks?artist=Artist+A", nil)
	req.SetPathValue("name", "Greatest Hits")
	rec := httptest.NewRecorder()
	s.handleAlbumTracks(rec, reqAs(user.ID, req))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var tracks []store.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(tracks) != 1 || tracks[0].Artist != "Artist A" {
		t.Fatalf("tracks = %+v, want exactly one Artist A track", tracks)
	}

	// Without ?artist=, both artists' same-titled-album tracks come back.
	req = httptest.NewRequest(http.MethodGet, "/api/albums/x/tracks", nil)
	req.SetPathValue("name", "Greatest Hits")
	rec = httptest.NewRecorder()
	s.handleAlbumTracks(rec, reqAs(user.ID, req))

	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("len(tracks) = %d, want 2 (union across artists)", len(tracks))
	}
}

func TestHandleListAlbums_ArtworkRepresentative(t *testing.T) {
	s, db, _, user := testServer(t)

	insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist C", Album: "Mixed Artwork", Title: "No Art"})
	withArt := insertTestTrack(t, s, db, user.ID, store.NewTrack{Artist: "Artist C", Album: "Mixed Artwork", Title: "Has Art", ArtworkExt: ".png"})

	req := httptest.NewRequest(http.MethodGet, "/api/albums", nil)
	rec := httptest.NewRecorder()
	s.handleListAlbums(rec, reqAs(user.ID, req))

	var albums []store.AlbumSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &albums); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	var found *store.AlbumSummary
	for i := range albums {
		if albums[i].Album == "Mixed Artwork" {
			found = &albums[i]
		}
	}
	if found == nil {
		t.Fatal("Mixed Artwork album not found in response")
	}
	if !found.HasArtwork {
		t.Error("HasArtwork = false, want true")
	}
	if found.RepresentativeTrackID != withArt.ID {
		t.Errorf("RepresentativeTrackID = %d, want %d (the artwork-bearing track)", found.RepresentativeTrackID, withArt.ID)
	}
}
