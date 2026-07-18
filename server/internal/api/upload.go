package api

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/Austycartcartt/mu3ic/server/internal/library"
)

const maxUploadSize = 200 << 20 // 200 MB, covers long FLAC files

// handleUpload streams a single audio file from a multipart request into
// a temp file, then hands it off to library.IngestFile — the handler
// itself only deals with HTTP/multipart concerns (size limits, field
// name, content-type sniffing); everything about turning a file on disk
// into a library track lives in IngestFile.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	mr, err := r.MultipartReader()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	part, err := findPart(mr, "audio")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer part.Close()

	declared := part.Header.Get("Content-Type")

	var sniffBuf [512]byte
	n, err := io.ReadFull(part, sniffBuf[:])
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		writeUploadReadError(w, err)
		return
	}
	sniffed := http.DetectContentType(sniffBuf[:n])

	if !strings.HasPrefix(declared, "audio/") && !strings.HasPrefix(sniffed, "audio/") {
		writeJSONError(w, http.StatusBadRequest, "file does not appear to be audio")
		return
	}

	tmpFile, err := os.CreateTemp(s.cfg.TempDir(), "upload-*")
	if err != nil {
		s.logger.Error("creating temp file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // best-effort; no-op once IngestFile renames the file away on success

	if _, err := tmpFile.Write(sniffBuf[:n]); err != nil {
		tmpFile.Close()
		s.logger.Error("writing temp file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
		return
	}
	if _, err := io.Copy(tmpFile, part); err != nil {
		tmpFile.Close()
		writeUploadReadError(w, err)
		return
	}
	if err := tmpFile.Close(); err != nil {
		s.logger.Error("closing temp file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process upload")
		return
	}

	track, err := library.IngestFile(r.Context(), s.store, s.cfg, tmpPath, part.FileName(), declared)
	if err != nil {
		s.logger.Error("ingesting file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save track")
		return
	}

	writeJSON(w, http.StatusCreated, track)
}

// findPart scans a multipart request for the first part with the given
// field name, closing (and discarding) any others along the way.
func findPart(mr *multipart.Reader, fieldName string) (*multipart.Part, error) {
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, errors.New(`missing "` + fieldName + `" field`)
		}
		if err != nil {
			return nil, errors.New("invalid multipart form")
		}
		if p.FormName() == fieldName {
			return p, nil
		}
		p.Close()
	}
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
