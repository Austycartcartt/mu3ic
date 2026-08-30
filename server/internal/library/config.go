package library

import "path/filepath"

// Config holds the local directory the upload handler stages files in
// before handing them to Storage. It's separate from Storage because
// staging is always local (the multipart body is streamed to a real file
// so dhowden/tag has an io.ReadSeeker) even when Storage is a remote
// object store.
type Config struct {
	LibraryDir string // upload staging lives at LibraryDir/tmp; the fs Storage backend also keeps objects directly under LibraryDir
}

// TempDir is where uploads are staged before being handed to Storage.Put.
// For the filesystem Storage backend it must be on the same filesystem as
// the library dir so that backend's temp-then-rename stays atomic, which
// is why it's a subdirectory rather than the OS default temp dir.
func (c Config) TempDir() string {
	return filepath.Join(c.LibraryDir, "tmp")
}
