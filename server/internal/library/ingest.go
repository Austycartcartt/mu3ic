package library

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Austycartcartt/mu3ic/server/internal/store"
)

// IngestFile takes a file already on local disk, extracts its metadata,
// moves it into the library under a new UUID storage key, and inserts a
// track row. It is the single entry point for adding music to the
// library — the HTTP upload handler is one caller; a future watch-folder
// scanner will be another. Contains no http.Request or other HTTP types,
// so it's callable directly from a test with just a path.
//
// Overrides carries caller-supplied title/artist/album/albumArtist values
// (e.g. from the upload preview screen's filename-parsed, hand-editable
// guess). A non-empty (after trimming) field replaces whatever the
// tag-extraction/filename fallback chain below would have produced for
// it; an empty field leaves that chain's result untouched. AlbumArtist has
// no filename-derived guess (see app/src/utils/parseFilename.ts) — it's
// populated from the ALBUMARTIST/TPE2 tag if present, or left for
// store.InsertTrack's per-track-artist fallback otherwise; set it
// explicitly (e.g. to "Various Artists") to group a compilation's
// differently-artist-tagged tracks under one album.
type Overrides struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
}

// declaredMimeType is the caller's best guess at the file's Content-Type
// (e.g. from a multipart part header, or "" for a caller with no such
// signal) — used as the primary source for the stored mime type, since
// content sniffing alone misclassifies several audio formats (see
// resolveMimeType).
func IngestFile(ctx context.Context, st *store.Store, cfg Config, srcPath, originalFilename, declaredMimeType string, overrides Overrides) (store.Track, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return store.Track{}, fmt.Errorf("opening source file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return store.Track{}, fmt.Errorf("statting source file: %w", err)
	}

	var sniffBuf [512]byte
	n, err := f.Read(sniffBuf[:])
	if err != nil && n == 0 {
		f.Close()
		return store.Track{}, fmt.Errorf("sniffing source file: %w", err)
	}
	sniffedType := http.DetectContentType(sniffBuf[:n])
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return store.Track{}, fmt.Errorf("rewinding source file: %w", err)
	}
	mimeType := resolveMimeType(declaredMimeType, sniffedType, originalFilename)

	md, err := ExtractMetadata(f)
	if err != nil {
		f.Close()
		return store.Track{}, fmt.Errorf("extracting metadata: %w", err)
	}
	f.Close()

	title := md.Title
	if title == "" {
		title = strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename))
	}
	artist := md.Artist
	if artist == "" {
		artist = "Unknown"
	}
	album := md.Album
	if album == "" {
		album = "Unknown"
	}
	// No "Unknown" fallback: an empty album artist means store.InsertTrack
	// falls back to this track's own artist, which is what a normal
	// (non-compilation) album wants.
	albumArtist := md.AlbumArtist

	if v := strings.TrimSpace(overrides.Title); v != "" {
		title = v
	}
	if v := strings.TrimSpace(overrides.Artist); v != "" {
		artist = v
	}
	if v := strings.TrimSpace(overrides.Album); v != "" {
		album = v
	}
	if v := strings.TrimSpace(overrides.AlbumArtist); v != "" {
		albumArtist = v
	}

	storageKey, err := NewUUID()
	if err != nil {
		return store.Track{}, fmt.Errorf("generating storage key: %w", err)
	}

	destPath := filepath.Join(cfg.LibraryDir, storageKey)
	if err := os.Rename(srcPath, destPath); err != nil {
		return store.Track{}, fmt.Errorf("moving file into library: %w", err)
	}

	// Artwork is a nice-to-have: a failure writing it shouldn't fail the
	// whole upload, consistent with ExtractMetadata's treatment of
	// missing/unreadable tags as a normal case, not an error.
	artworkExt := ""
	if md.Artwork != nil {
		artworkPath := filepath.Join(cfg.LibraryDir, storageKey+md.Artwork.Ext)
		if err := os.WriteFile(artworkPath, md.Artwork.Data, 0o644); err != nil {
			slog.Warn("writing artwork file", "storage_key", storageKey, "error", err)
		} else {
			artworkExt = md.Artwork.Ext
		}
	}

	track, err := st.InsertTrack(ctx, store.NewTrack{
		StorageKey:       storageKey,
		MimeType:         mimeType,
		OriginalFilename: originalFilename,
		Size:             info.Size(),
		Title:            title,
		Artist:           artist,
		Album:            album,
		AlbumArtist:      albumArtist,
		ArtworkExt:       artworkExt,
	})
	if err != nil {
		os.Remove(destPath)
		if artworkExt != "" {
			os.Remove(filepath.Join(cfg.LibraryDir, storageKey+artworkExt))
		}
		return store.Track{}, fmt.Errorf("inserting track: %w", err)
	}

	return track, nil
}

// extensionMimeTypes covers audio formats that either aren't in
// http.DetectContentType's sniff table (FLAC) or get sniffed to
// something other than an audio/* type (M4A sniffs as video/mp4's
// generic "ftyp" box; OGG sniffs as application/ogg).
var extensionMimeTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".ogg":  "audio/ogg",
	".wav":  "audio/wave",
	".aac":  "audio/aac",
	".opus": "audio/opus",
}

// resolveMimeType picks the mime type to store for an ingested file.
// Preferring the declared type (from a multipart Content-Type header)
// avoids relying solely on content sniffing, which is wrong for several
// common audio formats; the extension map is a last-resort fallback for
// callers (like a future bulk importer) with no declared type at all.
// mime.TypeByExtension isn't used here since it depends on the host's
// /etc/mime.types being present and correct, which isn't guaranteed
// across dev machines.
func resolveMimeType(declared, sniffed, originalFilename string) string {
	if strings.HasPrefix(declared, "audio/") {
		return declared
	}
	if strings.HasPrefix(sniffed, "audio/") {
		return sniffed
	}
	if mt, ok := extensionMimeTypes[strings.ToLower(filepath.Ext(originalFilename))]; ok {
		return mt
	}
	return "application/octet-stream"
}
