package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MartinAkopyan/go-users-api/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "user already exists")
	} else {
		h.logger.Error("service error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
