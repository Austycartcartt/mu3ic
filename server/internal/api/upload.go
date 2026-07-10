package api

import "net/http"

const (
	maxUploadSize = 1 << 30 // 1 GiB per file
	maxFormMemory = 32 << 20
)

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxFormMemory); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, `missing "file" field`)
		return
	}
	defer file.Close()

	track, err := s.store.InsertTrack(r.Context(), header.Filename, header.Size)
	if err != nil {
		s.logger.Error("inserting track", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save track")
		return
	}

	if err := s.storage.Save(track.ID, track.Filename, file); err != nil {
		s.logger.Error("saving track file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save track file")
		return
	}

	writeJSON(w, http.StatusCreated, track)
}
