// Package library handles audio file storage on disk.
package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Storage is kept small and swappable per PROJECT.md: only a filesystem
// implementation exists today, but MinIO/S3 support can be added later
// behind this same interface without touching callers.
type Storage interface {
	Save(id int64, filename string, r io.Reader) error
	Open(id int64, filename string) (*os.File, error)
}

// FileStorage stores each track's file under dir/<id>/<filename>.
type FileStorage struct {
	dir string
}

func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir %q: %w", dir, err)
	}
	return &FileStorage{dir: dir}, nil
}

func (f *FileStorage) trackDir(id int64) string {
	return filepath.Join(f.dir, strconv.FormatInt(id, 10))
}

func (f *FileStorage) Save(id int64, filename string, r io.Reader) error {
	dir := f.trackDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating track dir: %w", err)
	}

	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, r); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

func (f *FileStorage) Open(id int64, filename string) (*os.File, error) {
	file, err := os.Open(filepath.Join(f.trackDir(id), filename))
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	return file, nil
}
