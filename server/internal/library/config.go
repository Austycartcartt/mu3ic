package library

import "path/filepath"

// Config holds filesystem paths IngestFile needs. Kept separate from
// Storage (the swappable read-side abstraction) since the atomic
// os.Rename in IngestFile is inherently tied to a local filesystem.
type Config struct {
	LibraryDir string // audio files land at LibraryDir/<uuid>
}

// TempDir is where uploads are staged before being renamed into
// LibraryDir. It must be on the same filesystem as LibraryDir so the
// rename is atomic, which is why it's a subdirectory rather than the OS
// default temp dir.
func (c Config) TempDir() string {
	return filepath.Join(c.LibraryDir, "tmp")
}
