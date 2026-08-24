package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Track mirrors the tracks table. JSON tags define the wire format
// returned to clients. StorageKey, MimeType, and ArtworkExt are internal
// details the API handlers need (to open the right file / set the right
// Content-Type) but that clients have no use for, so they're excluded
// from the JSON response; HasArtwork is derived from ArtworkExt instead
// of exposing the extension itself.
type Track struct {
	ID               int64     `json:"id"`
	StorageKey       string    `json:"-"`
	MimeType         string    `json:"-"`
	OriginalFilename string    `json:"original_filename"`
	Size             int64     `json:"size"`
	Title            string    `json:"title"`
	Artist           string    `json:"artist"`
	Album            string    `json:"album"`
	AlbumArtist      string    `json:"-"` // internal grouping key for ListAlbums; see store.AlbumSummary
	DurationSeconds  *int      `json:"duration_seconds"` // always nil this phase; TODO(phase-later): decode duration
	ArtworkExt       string    `json:"-"`
	HasArtwork       bool      `json:"hasArtwork"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

// NewTrack is the set of fields IngestFile has in hand once it's ready to
// insert a row: the UUID storage key, resolved mime type, extracted/
// fallback-applied metadata, and (if present) the artwork file extension.
type NewTrack struct {
	StorageKey       string
	MimeType         string
	OriginalFilename string
	Size             int64
	Title            string
	Artist           string
	Album            string
	AlbumArtist      string // "" means unset — InsertTrack falls back to Artist
	ArtworkExt       string // "" if the track has no artwork
}

// dbTrack mirrors the tracks table exactly, including the columns that
// are nullable (duration is never extracted this phase; artwork is only
// present for some tracks).
type dbTrack struct {
	ID               int64
	StorageKey       string
	MimeType         string
	OriginalFilename string
	Size             int64
	Title            string
	Artist           string
	Album            string
	AlbumArtist      string
	DurationSeconds  sql.NullInt32
	ArtworkExt       sql.NullString
	UploadedAt       time.Time
}

func (d dbTrack) toTrack() Track {
	var duration *int
	if d.DurationSeconds.Valid {
		v := int(d.DurationSeconds.Int32)
		duration = &v
	}
	return Track{
		ID:               d.ID,
		StorageKey:       d.StorageKey,
		MimeType:         d.MimeType,
		OriginalFilename: d.OriginalFilename,
		Size:             d.Size,
		Title:            d.Title,
		Artist:           d.Artist,
		Album:            d.Album,
		AlbumArtist:      d.AlbumArtist,
		DurationSeconds:  duration,
		ArtworkExt:       d.ArtworkExt.String,
		HasArtwork:       d.ArtworkExt.Valid && d.ArtworkExt.String != "",
		UploadedAt:       d.UploadedAt,
	}
}

const trackColumns = `id, storage_key, mime_type, original_filename, size, title, artist, album, album_artist, duration_seconds, artwork_ext, uploaded_at`

func scanTrack(row interface{ Scan(...any) error }) (Track, error) {
	var d dbTrack
	err := row.Scan(&d.ID, &d.StorageKey, &d.MimeType, &d.OriginalFilename, &d.Size,
		&d.Title, &d.Artist, &d.Album, &d.AlbumArtist, &d.DurationSeconds, &d.ArtworkExt, &d.UploadedAt)
	if err != nil {
		return Track{}, err
	}
	return d.toTrack(), nil
}

// InsertTrack records a newly ingested file's storage key, resolved mime
// type, and extracted (or fallback) metadata. The id is assigned by
// Postgres (BIGSERIAL), so there's no need to generate one in Go.
func (s *Store) InsertTrack(ctx context.Context, nt NewTrack) (Track, error) {
	// An unset album artist defaults to the track's own artist, so a
	// normal (non-compilation) album keeps grouping the way it always
	// has — only an explicit album artist (tag or override) collapses a
	// various-artists album into one row. See migrations/004_album_artist.sql.
	albumArtist := nt.AlbumArtist
	if albumArtist == "" {
		albumArtist = nt.Artist
	}

	row := s.db.QueryRowContext(ctx,
		`INSERT INTO tracks (storage_key, mime_type, original_filename, size, title, artist, album, album_artist, artwork_ext)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING `+trackColumns,
		nt.StorageKey, nt.MimeType, nt.OriginalFilename, nt.Size,
		nt.Title, nt.Artist, nt.Album, albumArtist, nullableString(nt.ArtworkExt),
	)
	track, err := scanTrack(row)
	if err != nil {
		return Track{}, fmt.Errorf("inserting track: %w", err)
	}
	return track, nil
}

// nullableString converts "" to a SQL NULL, so "no artwork" is stored as
// NULL rather than an empty-string sentinel.
func nullableString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func (s *Store) ListTracks(ctx context.Context) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+trackColumns+` FROM tracks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing tracks: %w", err)
	}
	defer rows.Close()

	tracks := []Track{} // not nil, so an empty result serializes as [] rather than null
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

func (s *Store) GetTrack(ctx context.Context, id int64) (Track, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+trackColumns+` FROM tracks WHERE id = $1`, id)
	track, err := scanTrack(row)
	if err != nil {
		return Track{}, fmt.Errorf("getting track %d: %w", id, err)
	}
	return track, nil
}
