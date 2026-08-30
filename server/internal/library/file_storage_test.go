package library

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStorage_PutOpenRoundTrip(t *testing.T) {
	s, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage: %v", err)
	}
	want := []byte("some audio bytes")

	if err := s.Put(context.Background(), "key-1", bytes.NewReader(want), int64(len(want)), "audio/mpeg"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	obj, err := s.Open(context.Background(), "key-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer obj.Close()

	got, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, want)
	}
}

func TestFileStorage_OpenMissingIsNotExist(t *testing.T) {
	s, _ := NewFileStorage(t.TempDir())
	_, err := s.Open(context.Background(), "nope")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Open(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestFileStorage_DeleteIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewFileStorage(dir)
	_ = s.Put(context.Background(), "k", strings.NewReader("x"), 1, "")

	if err := s.Delete(context.Background(), "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Deleting again, and deleting a key that never existed, must not error.
	if err := s.Delete(context.Background(), "k", "never-existed"); err != nil {
		t.Errorf("Delete(missing) error = %v, want nil", err)
	}
}

// errAfterN returns an error once n bytes have been read, to simulate a
// truncated upload stream.
type errAfterN struct {
	n int
}

func (e *errAfterN) Read(p []byte) (int, error) {
	if e.n <= 0 {
		return 0, errors.New("boom")
	}
	k := min(len(p), e.n)
	for i := range k {
		p[i] = 'a'
	}
	e.n -= k
	return k, nil
}

func TestFileStorage_PutLeavesNoPartialObjectOnReaderError(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewFileStorage(dir)

	err := s.Put(context.Background(), "half", &errAfterN{n: 8}, 64, "")
	if err == nil {
		t.Fatal("Put with a failing reader returned nil error")
	}

	if _, statErr := os.Stat(filepath.Join(dir, "half")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("a partial object was left at the real key: stat err = %v", statErr)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("temp files left behind after failed Put: %v", entries)
	}
}
