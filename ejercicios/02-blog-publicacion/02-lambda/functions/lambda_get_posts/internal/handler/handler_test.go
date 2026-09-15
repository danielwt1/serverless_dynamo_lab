package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"lambda_posts_mutable/internal/domain"
)

type getInfoStub struct {
	publishedAuthor, draftAuthor, cursor string
	limit                              int32
	err                                error
}

func (s *getInfoStub) FindPublishedPosts(_ context.Context, authorID, cursor string, limit int32) (domain.PostsPage, error) {
	s.publishedAuthor, s.cursor, s.limit = authorID, cursor, limit
	return domain.PostsPage{Items: []domain.PostModel{{PostId: "post-1"}}}, s.err
}

func (s *getInfoStub) FindDraftPosts(_ context.Context, authorID, cursor string, limit int32) (domain.PostsPage, error) {
	s.draftAuthor, s.cursor, s.limit = authorID, cursor, limit
	return domain.PostsPage{}, s.err
}

func requestWithAuthor(path, authorID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("authorId", authorID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestGetInfoHandler_GetPublishedPosts_UsesQueryParameters(t *testing.T) {
	stub := &getInfoStub{}
	response := httptest.NewRecorder()

	NewGetInfoHandler(stub).GetPublishedPosts(response, requestWithAuthor("/authors/author-1/posts?limit=25&cursor=next", "author-1"))

	if response.Code != http.StatusOK || stub.publishedAuthor != "author-1" || stub.cursor != "next" || stub.limit != 25 {
		t.Errorf("status=%d stub=%+v", response.Code, stub)
	}
}

func TestGetInfoHandler_GetDraftPosts_UsesDefaultLimit(t *testing.T) {
	stub := &getInfoStub{}
	response := httptest.NewRecorder()

	NewGetInfoHandler(stub).GetDraftPosts(response, requestWithAuthor("/authors/author-1/drafts", "author-1"))

	if response.Code != http.StatusOK || stub.draftAuthor != "author-1" || stub.limit != defaultLimit {
		t.Errorf("status=%d stub=%+v", response.Code, stub)
	}
}

func TestGetInfoHandler_GetPublishedPosts_RejectsInvalidLimit(t *testing.T) {
	stub := &getInfoStub{}
	response := httptest.NewRecorder()

	NewGetInfoHandler(stub).GetPublishedPosts(response, requestWithAuthor("/authors/author-1/posts?limit=101", "author-1"))

	if response.Code != http.StatusBadRequest || stub.publishedAuthor != "" {
		t.Errorf("status=%d stub=%+v", response.Code, stub)
	}
}

func TestGetInfoHandler_GetPublishedPosts_MapsInvalidCursor(t *testing.T) {
	stub := &getInfoStub{err: domain.ErrInvalidCursor}
	response := httptest.NewRecorder()

	NewGetInfoHandler(stub).GetPublishedPosts(response, requestWithAuthor("/authors/author-1/posts", "author-1"))

	if response.Code != http.StatusBadRequest {
		t.Errorf("status=%d", response.Code)
	}
	if !errors.Is(stub.err, domain.ErrInvalidCursor) {
		t.Errorf("stub error = %v", stub.err)
	}
}
