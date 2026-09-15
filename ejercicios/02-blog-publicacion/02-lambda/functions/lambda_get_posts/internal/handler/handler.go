package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"lambda_posts_mutable/internal/domain"
	"lambda_posts_mutable/internal/domain/ports/in"

	"github.com/go-chi/chi/v5"
)

const defaultLimit int32 = 10

type GetInfoHandler struct {
	apiPort in.GetInfoApiPort
}

func NewGetInfoHandler(apiPort in.GetInfoApiPort) *GetInfoHandler {
	return &GetInfoHandler{
		apiPort: apiPort,
	}
}

// GetPublishedPosts handles GET /authors/{authorId}/posts?limit=10&cursor=...
func (h *GetInfoHandler) GetPublishedPosts(w http.ResponseWriter, r *http.Request) {
	authorID := strings.TrimSpace(chi.URLParam(r, "authorId"))
	if authorID == "" {
		writeError(w, http.StatusBadRequest, "authorId is required")
		return
	}

	limit, err := limitFromRequest(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	posts, err := h.apiPort.FindPublishedPosts(
		r.Context(),
		authorID,
		r.URL.Query().Get("cursor"),
		limit,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, posts)
}

// GetDraftPosts handles GET /authors/{authorId}/drafts?limit=10&cursor=...
func (h *GetInfoHandler) GetDraftPosts(w http.ResponseWriter, r *http.Request) {
	authorID := strings.TrimSpace(chi.URLParam(r, "authorId"))
	if authorID == "" {
		writeError(w, http.StatusBadRequest, "authorId is required")
		return
	}

	limit, err := limitFromRequest(r)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	posts, err := h.apiPort.FindDraftPosts(
		r.Context(),
		authorID,
		r.URL.Query().Get("cursor"),
		limit,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, posts)
}

func limitFromRequest(r *http.Request) (int32, error) {
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.ParseInt(rawLimit, 10, 32)
	if err != nil || limit < 1 || limit > 100 {
		return 0, domain.ErrInvalidLimit
	}

	return int32(limit), nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidLimit), errors.Is(err, domain.ErrInvalidCursor):
		writeError(w, http.StatusBadRequest, err.Error())
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

	if err := json.NewEncoder(w).Encode(value); err != nil {
		// At this point the HTTP status was already sent. The best action is to
		// let the Lambda adapter return the response that was written so far.
		return
	}
}
