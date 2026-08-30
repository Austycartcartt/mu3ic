// Package library handles audio and artwork storage.
package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Object is a readable, seekable, closeable handle to a stored object —
// exactly what http.ServeContent needs to serve a track with Range
// support.
type Object interface {
	io.ReadSeekCloser
}

// Storage is the swappable audio + artwork backend. Object keys are the
// bare track storage_key (audio) or storage_key+artwork_ext (artwork).
// FileStorage is used in local dev; in production it's an S3-compatible
// object store — NeonStorage (Neon Object Storage) or R2Storage
// (Cloudflare R2) — selected by the STORAGE_BACKEND env var.
//
// Storage owns writes as well as reads: IngestFile hands every ingested
// file to Put. An earlier revision kept this interface read-only and
// wrote audio with os.Rename inside IngestFile, which only worked for a
// local filesystem — see docs/DECISIONS.md.
type Storage interface {
	// Put stores exactly size bytes read from r under key. contentType is
	// kept as object metadata by backends that need it (object stores);
	// FileStorage ignores it, since a track's Content-Type is resolved
	// from the DB at serve time.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Open returns a handle for streaming an object's bytes. It returns an
	// error satisfying errors.Is(err, fs.ErrNotExist) when key has no
	// stored object.
	Open(ctx context.Context, key string) (Object, error)

	// Delete removes the named objects. A key with no stored object is not
	// an error.
	Delete(ctx context.Context, keys ...string) error
}

// Presigner is an optional Storage capability: a backend that can hand a
// client a direct, time-limited URL for an object so the bytes never
// transit this server. The stream and artwork handlers type-assert for it
// and 302-redirect when it's present, falling back to Open +
// http.ServeContent when it isn't (FileStorage).
type Presigner interface {
	PresignGet(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
}

// ArtworkContentType maps a stored artwork extension to its MIME type.
// ExtractMetadata only ever writes ".jpg" or ".png" (see metadata.go), so
// those are the only cases that matter; an unknown ext returns "".
func ArtworkContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return ""
	}
}

// FileStorage stores each object as a file at dir/<key>: audio at
// dir/<storage_key>, artwork at dir/<storage_key><ext>.
type FileStorage struct {
	dir string
}

func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating library dir %q: %w", dir, err)
	}
	return &FileStorage{dir: dir}, nil
}

// Put writes to a temp file in the same directory, then renames it into
// place, so a reader error or a crash mid-write never leaves a
// partially-written object at the real key. Same dir for temp and final
// means the rename is atomic. size and contentType are unused: on a
// filesystem the byte count is implicit and Content-Type comes from the
// DB at serve time.
func (f *FileStorage) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	tmp, err := os.CreateTemp(f.dir, ".put-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename below succeeds

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return fmt.Errorf("writing object: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing object: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(f.dir, key)); err != nil {
		return fmt.Errorf("moving object into place: %w", err)
	}
	return nil
}

func (f *FileStorage) Open(_ context.Context, key string) (Object, error) {
	file, err := os.Open(filepath.Join(f.dir, key))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("opening %q: %w", key, fs.ErrNotExist)
		}
		return nil, fmt.Errorf("opening %q: %w", key, err)
	}
	return file, nil
}

func (f *FileStorage) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		if err := os.Remove(filepath.Join(f.dir, key)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("deleting %q: %w", key, err)
		}
	}
	return nil
}
