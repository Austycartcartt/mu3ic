package store

import (
	"context"
	"fmt"
)

// AlbumSummary is a grouped view over (tracks.album, tracks.album_artist).
// It's grouped by the pair, not album alone, because album titles aren't
// unique across artists (e.g. two different artists each with a "Greatest
// Hits") — grouping by album alone would incorrectly merge them into one
// row. It uses album_artist rather than each track's own artist so that a
// various-artists compilation (one album, many different track artists,
// sharing one album_artist) still collapses into a single row instead of
// one row per track artist.
type AlbumSummary struct {
	Album                 string `json:"album"`
	Artist                string `json:"artist"` // sourced from album_artist, not any one track's artist
	TrackCount            int    `json:"track_count"`
	RepresentativeTrackID int64  `json:"representative_track_id"`
	HasArtwork            bool   `json:"hasArtwork"`
}

func (s *Store) ListAlbums(ctx context.Context, userID int64) ([]AlbumSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT album, album_artist, COUNT(*) AS track_count,
		        (ARRAY_AGG(id ORDER BY (artwork_ext IS NOT NULL) DESC, id))[1] AS representative_track_id,
		        BOOL_OR(artwork_ext IS NOT NULL) AS has_artwork
		 FROM tracks
		 WHERE user_id = $1
		 GROUP BY album, album_artist
		 ORDER BY album, album_artist`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing albums: %w", err)
	}
	defer rows.Close()

	albums := []AlbumSummary{} // not nil, so an empty result serializes as [] rather than null
	for rows.Next() {
		var a AlbumSummary
		if err := rows.Scan(&a.Album, &a.Artist, &a.TrackCount, &a.RepresentativeTrackID, &a.HasArtwork); err != nil {
			return nil, fmt.Errorf("scanning album: %w", err)
		}
		albums = append(albums, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating albums: %w", err)
	}
	return albums, nil
}

// ListTracksByAlbum orders by track_number (untagged tracks, with a NULL
// track_number, sort after any tagged track) then title as a tiebreaker/
// fallback for untagged albums. An empty albumArtist means no disambiguator
// was given (matches any album artist); pass the AlbumSummary.Artist the
// caller has, since album titles aren't unique across artists. Matched
// against album_artist, not each track's own artist, so a various-artists
// compilation's tracks (each with a different artist, sharing one
// album_artist) all come back together.
func (s *Store) ListTracksByAlbum(ctx context.Context, userID int64, album string, albumArtist string) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+trackColumns+` FROM tracks
		 WHERE user_id = $1 AND album = $2 AND ($3 = '' OR album_artist = $3)
		 ORDER BY track_number NULLS LAST, title`, userID, album, albumArtist)
	if err != nil {
		return nil, fmt.Errorf("listing tracks for album %q: %w", album, err)
	}
	defer rows.Close()

	tracks := []Track{}
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning track: %w", err)
		}
		tracks = append(tracks, track)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tracks: %w", err)
	}
	return tracks, nil
}
