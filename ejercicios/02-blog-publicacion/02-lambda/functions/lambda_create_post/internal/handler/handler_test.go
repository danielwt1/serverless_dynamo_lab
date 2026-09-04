package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"lambda_create_post/internal/domain"
)

type createPostStub struct {
	authorID, description string
	status                domain.PostStatus
	err                   error
}

func (s *createPostStub) CreatePost(_ context.Context, authorID, description string, status domain.PostStatus) (domain.PostModel, error) {
	s.authorID, s.description, s.status = authorID, description, status
	if s.err != nil {
		return domain.PostModel{}, s.err
	}
	return domain.PostModel{PostID: "post-1", AuthorID: authorID, Description: description, Status: status}, nil
}

func TestCreatePostHandler_CreatePost_CreatesPost(t *testing.T) {
	stub := &createPostStub{}
	req := httptest.NewRequest(http.MethodPost, "/authors/author-1/posts", strings.NewReader(`{"description":"hello","status":"DRAFT"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("authorId", "author-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	response := httptest.NewRecorder()

	NewCreatePostHandler(stub).CreatePost(response, req)

	if response.Code != http.StatusCreated || stub.authorID != "author-1" || stub.description != "hello" || stub.status != domain.PostStatusDraft {
		t.Errorf("status=%d stub=%+v body=%s", response.Code, stub, response.Body.String())
	}
}

func TestCreatePostHandler_CreatePost_RejectsInvalidInput(t *testing.T) {
	stub := &createPostStub{}
	req := httptest.NewRequest(http.MethodPost, "/authors/author-1/posts", strings.NewReader(`{"unknown":true}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("authorId", "author-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	response := httptest.NewRecorder()

	NewCreatePostHandler(stub).CreatePost(response, req)

	if response.Code != http.StatusBadRequest || stub.authorID != "" {
		t.Errorf("status=%d stub=%+v", response.Code, stub)
	}
}

func TestCreatePostHandler_CreatePost_MapsDomainError(t *testing.T) {
	stub := &createPostStub{err: domain.ErrPostAlreadyExists}
	req := httptest.NewRequest(http.MethodPost, "/authors/author-1/posts", strings.NewReader(`{"description":"hello","status":"DRAFT"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("authorId", "author-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	response := httptest.NewRecorder()

	NewCreatePostHandler(stub).CreatePost(response, req)

	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), domain.ErrPostAlreadyExists.Error()) {
		t.Errorf("status=%d body=%s", response.Code, response.Body.String())
	}
}
