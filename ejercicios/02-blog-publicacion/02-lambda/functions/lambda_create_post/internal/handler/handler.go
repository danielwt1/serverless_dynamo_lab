package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"lambda_create_post/internal/domain"
	"lambda_create_post/internal/domain/ports/in"
)

type CreatePostHandler struct {
	apiPort in.CreatePostAPIPort
}

type createPostRequest struct {
	Description string            `json:"description"`
	Status      domain.PostStatus `json:"status"`
}

func NewCreatePostHandler(apiPort in.CreatePostAPIPort) *CreatePostHandler {
	return &CreatePostHandler{apiPort: apiPort}
}

func (h *CreatePostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	authorID := strings.TrimSpace(chi.URLParam(r, "authorId"))
	if authorID == "" {
		writeError(w, http.StatusBadRequest, domain.ErrInvalidAuthorID.Error())
		return
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var request createPostRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	post, err := h.apiPort.CreatePost(r.Context(), authorID, request.Description, request.Status)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidAuthorID), errors.Is(err, domain.ErrInvalidDescription), errors.Is(err, domain.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrPostAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
