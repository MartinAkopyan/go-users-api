package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		h.logger.Warn("decode failed", "error", err)
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	err = h.validate.Struct(req)

	if err != nil {
		var verrs validator.ValidationErrors
		if !errors.As(err, &verrs) {
			h.handleServiceError(w, err)
			return
		}

		fields := make(map[string]string)

		for _, fe := range verrs {
			fields[fe.Field()] = fe.Tag()
		}

		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "validation failed",
			"fields": fields,
		})
		return
	}

	if len(req.Password) > 72 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "validation failed",
			"fields": map[string]string{"password": "max"},
		})
		return
	}

	u, err := h.svc.Register(r.Context(), req.FullName, req.Email, req.Password)

	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toUserResponse(u))
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	uID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	u, err := h.svc.GetByID(r.Context(), uID)

	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(u))
}
