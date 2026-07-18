package library

import (
	"io"
	"log/slog"

	"github.com/dhowden/tag"
)

// Metadata holds tags extracted from an audio file. Zero values (empty
// string, 0) mean the tag was absent — a file with no tags is still a
// valid track, so callers must not treat these as errors.
type Metadata struct {
	Title       string
	Artist      string
	Album       string
	TrackNumber int
	Year        int
	Artwork     *Artwork // nil if the file has no usable embedded picture
}

// Artwork is embedded cover art pulled from a tag frame/atom.
type Artwork struct {
	Data []byte
	Ext  string // ".jpg" or ".png", derived from the picture's MIME type
}

// ExtractMetadata reads embedded tags from an audio file. A file that
// tag.ReadFrom cannot parse (no tags, or an unsupported/corrupt format) is
// not an upload failure: it's logged as a warning and Metadata is returned
// zeroed, so the caller falls back to the filename as the display title.
func ExtractMetadata(r io.ReadSeeker) (Metadata, error) {
	t, err := tag.ReadFrom(r)
	if err != nil {
		slog.Warn("no readable tags in uploaded file", "error", err)
		return Metadata{}, nil
	}

	md := Metadata{
		Title:  t.Title(),
		Artist: t.Artist(),
		Album:  t.Album(),
		Year:   t.Year(),
	}
	if trackNum, _ := t.Track(); trackNum > 0 {
		md.TrackNumber = trackNum
	}

	if pic := t.Picture(); pic != nil {
		if ext := extForMIMEType(pic.MIMEType); ext != "" {
			md.Artwork = &Artwork{Data: pic.Data, Ext: ext}
		}
	}

	return md, nil
}

// extForMIMEType maps the picture MIME types dhowden/tag normalizes to
// across ID3v2, MP4, and Vorbis comments onto a file extension. Anything
// else (e.g. image/gif) is ignored per the Phase 2 spec.
func extForMIMEType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}
