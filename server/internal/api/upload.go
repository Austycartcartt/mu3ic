package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
)

const maxUploadSize = 200 << 20 // 200 MB, covers long FLAC files

// handleUpload streams a single audio file from a multipart request into
// a temp file, then hands it off to library.IngestFile — the handler
// itself only deals with HTTP/multipart concerns (size limits, field
// name, content-type sniffing); everything about turning a file on disk
// into a library track lives in IngestFile.
//
// Besides the required "audio" part, it accepts optional "title"/
// "artist"/"album"/"album_artist"/"track_number" text parts — the upload
// preview screen's filename-parsed, hand-editable guess — passed through to
// IngestFile as overrides.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	mr, err := r.MultipartReader()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	var (
		tmpPath          string
		originalFilename string
		declared         string
		overrides        library.Overrides
		sawAudio         bool
	)
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath) // best-effort; no-op once IngestFile renames the file away on success
		}
	}()

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}

		switch part.FormName() {
		case "audio":
			sawAudio = true
			originalFilename = part.FileName()
			declared = part.Header.Get("Content-Type")

			var sniffBuf [512]byte
			n, err := io.ReadFull(part, sniffBuf[:])
			if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
				part.Close()
				writeUploadReadError(w, err)
				return
			}
			sniffed := http.DetectContentType(sniffBuf[:n])

			if !strings.HasPrefix(declared, "audio/") && !strings.HasPrefix(sniffed, "audio/") {
				part.Close()
				writeJSONError(w, http.StatusBadRequest, "file does not appear to be audio")
				return
			}

			tmpFile, err := os.CreateTemp(s.cfg.TempDir(), "upload-*")
			if err != nil {
				part.Close()
				s.logger.Error("creating temp file", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
				return
			}
			tmpPath = tmpFile.Name()

			if _, err := tmpFile.Write(sniffBuf[:n]); err != nil {
				tmpFile.Close()
				part.Close()
				s.logger.Error("writing temp file", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
				return
			}
			if _, err := io.Copy(tmpFile, part); err != nil {
				tmpFile.Close()
				part.Close()
				writeUploadReadError(w, err)
				return
			}
			if err := tmpFile.Close(); err != nil {
				part.Close()
				s.logger.Error("closing temp file", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
				return
			}
		case "title", "artist", "album", "album_artist", "track_number":
			value, err := io.ReadAll(part)
			if err != nil {
				part.Close()
				writeUploadReadError(w, err)
				return
			}
			switch part.FormName() {
			case "title":
				overrides.Title = string(value)
			case "artist":
				overrides.Artist = string(value)
			case "album":
				overrides.Album = string(value)
			case "album_artist":
				overrides.AlbumArtist = string(value)
			case "track_number":
				// A malformed value (shouldn't happen from the app's own
				// numeric input) just leaves the override unset rather than
				// failing the whole upload.
				if n, err := strconv.Atoi(strings.TrimSpace(string(value))); err == nil {
					overrides.TrackNumber = n
				}
			}
		}
		part.Close()
	}

	if !sawAudio {
		writeJSONError(w, http.StatusBadRequest, `missing "audio" field`)
		return
	}

	track, err := library.IngestFile(r.Context(), s.store, s.cfg, tmpPath, originalFilename, declared, overrides)
	if err != nil {
		s.logger.Error("ingesting file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save track")
		return
	}

	writeJSON(w, http.StatusCreated, track)
}

// writeUploadReadError distinguishes a size-limit violation (413) from
// any other read failure (500) when copying the uploaded part.
func writeUploadReadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
}
