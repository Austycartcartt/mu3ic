// Package library handles audio file storage on disk.
package library

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Storage is kept small and swappable per PROJECT.md: only a filesystem
// implementation exists today, but MinIO/S3 support can be added later
// behind this same interface without touching callers. Writes (both
// audio and artwork) happen inside IngestFile, not through Storage —
// Storage is read-only because the atomic rename-into-place step is
// inherently part of the ingest pipeline, not a generic "save" operation.
type Storage interface {
	// Open opens the audio file for a track by its storage key.
	Open(storageKey string) (*os.File, error)

	// ArtworkPath resolves the on-disk artwork file for a track, given its
	// storage key and known extension (from the DB), for use with
	// http.ServeFile. It returns an error satisfying
	// errors.Is(err, fs.ErrNotExist) if the file is missing or ext is "".
	ArtworkPath(storageKey, ext string) (string, error)
}

// FileStorage stores each track's audio at dir/<storage_key> and, if
// present, its artwork as a sibling file at dir/<storage_key><ext>.
type FileStorage struct {
	dir string
}

func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating library dir %q: %w", dir, err)
	}
	return &FileStorage{dir: dir}, nil
}

func (f *FileStorage) Open(storageKey string) (*os.File, error) {
	file, err := os.Open(filepath.Join(f.dir, storageKey))
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return file, nil
}

func (f *FileStorage) ArtworkPath(storageKey, ext string) (string, error) {
	if ext == "" {
		return "", fmt.Errorf("artwork for %q: %w", storageKey, fs.ErrNotExist)
	}
	path := filepath.Join(f.dir, storageKey+ext)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("artwork for %q: %w", storageKey, fs.ErrNotExist)
		}
		return "", fmt.Errorf("stat artwork for %q: %w", storageKey, err)
	}
	return path, nil
}
