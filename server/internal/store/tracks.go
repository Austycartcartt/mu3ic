package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	TrackNumber      *int      `json:"track_number"` // nil if untagged
	DurationSeconds  *int      `json:"duration_seconds"` // always nil this phase; TODO(phase-later): decode duration
	ArtworkExt       string    `json:"-"`
	HasArtwork       bool      `json:"hasArtwork"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

// NewTrack is the set of fields IngestFile has in hand once it's ready to
// insert a row: the owning user, UUID storage key, resolved mime type,
// extracted/fallback-applied metadata, and (if present) the artwork file
// extension.
type NewTrack struct {
	UserID           int64
	StorageKey       string
	MimeType         string
	OriginalFilename string
	Size             int64
	Title            string
	Artist           string
	Album            string
	AlbumArtist      string // "" means unset — InsertTrack falls back to Artist
	TrackNumber      int    // 0 means untagged/unknown — stored as NULL
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
	TrackNumber      sql.NullInt32
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
	var trackNumber *int
	if d.TrackNumber.Valid {
		v := int(d.TrackNumber.Int32)
		trackNumber = &v
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
		TrackNumber:      trackNumber,
		DurationSeconds:  duration,
		ArtworkExt:       d.ArtworkExt.String,
		HasArtwork:       d.ArtworkExt.Valid && d.ArtworkExt.String != "",
		UploadedAt:       d.UploadedAt,
	}
}

const trackColumns = `id, storage_key, mime_type, original_filename, size, title, artist, album, album_artist, track_number, duration_seconds, artwork_ext, uploaded_at`

// prefixedTrackColumns is trackColumns with every column qualified by the
// given table alias, for SELECTs that JOIN tracks against another table
// with a colliding column name (e.g. playlists.id). Column order still
// matches scanTrack.
func prefixedTrackColumns(alias string) string {
	cols := strings.Split(trackColumns, ", ")
	for i, c := range cols {
		cols[i] = alias + "." + c
	}
	return strings.Join(cols, ", ")
}

func scanTrack(row interface{ Scan(...any) error }) (Track, error) {
	var d dbTrack
	err := row.Scan(&d.ID, &d.StorageKey, &d.MimeType, &d.OriginalFilename, &d.Size,
		&d.Title, &d.Artist, &d.Album, &d.AlbumArtist, &d.TrackNumber, &d.DurationSeconds, &d.ArtworkExt, &d.UploadedAt)
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
		`INSERT INTO tracks (user_id, storage_key, mime_type, original_filename, size, title, artist, album, album_artist, track_number, artwork_ext)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING `+trackColumns,
		nt.UserID, nt.StorageKey, nt.MimeType, nt.OriginalFilename, nt.Size,
		nt.Title, nt.Artist, nt.Album, albumArtist, nullableTrackNumber(nt.TrackNumber), nullableString(nt.ArtworkExt),
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

// nullableTrackNumber converts 0 (ExtractMetadata's "no tag" value) to a
// SQL NULL, so an untagged track sorts after any real track number instead
// of colliding with track 0.
func nullableTrackNumber(n int) sql.NullInt32 {
	return sql.NullInt32{Int32: int32(n), Valid: n > 0}
}

func (s *Store) ListTracks(ctx context.Context, userID int64) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+trackColumns+` FROM tracks WHERE user_id = $1 ORDER BY id`, userID)
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

// likeEscaper neutralizes the LIKE/ILIKE wildcards so a query containing
// '%' or '_' is matched literally. Paired with an `ESCAPE '\'` clause on
// the query itself.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SearchTracks does a case-insensitive substring match of query against
// each track's title, artist, and album, scoped to userID. Deliberately a
// plain ILIKE '%q%' rather than Postgres full-text search — no stemming or
// ranking, but zero new machinery and fine at a personal library's scale
// (see the Phase 6 entry in docs/DECISIONS.md). The caller is expected to
// have trimmed query and to skip the call entirely when it's empty.
func (s *Store) SearchTracks(ctx context.Context, userID int64, query string) ([]Track, error) {
	pattern := "%" + likeEscaper.Replace(query) + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+trackColumns+` FROM tracks
		 WHERE user_id = $1
		   AND (title ILIKE $2 ESCAPE '\' OR artist ILIKE $2 ESCAPE '\' OR album ILIKE $2 ESCAPE '\')
		 ORDER BY title
		 LIMIT 100`, userID, pattern)
	if err != nil {
		return nil, fmt.Errorf("searching tracks: %w", err)
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

// GetTrack scopes the lookup to userID, so requesting another user's
// track id is indistinguishable from requesting one that doesn't exist —
// both return sql.ErrNoRows, which the stream/artwork handlers map to 404.
func (s *Store) GetTrack(ctx context.Context, id, userID int64) (Track, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+trackColumns+` FROM tracks WHERE id = $1 AND user_id = $2`, id, userID)
	track, err := scanTrack(row)
	if err != nil {
		return Track{}, fmt.Errorf("getting track %d: %w", id, err)
	}
	return track, nil
}
