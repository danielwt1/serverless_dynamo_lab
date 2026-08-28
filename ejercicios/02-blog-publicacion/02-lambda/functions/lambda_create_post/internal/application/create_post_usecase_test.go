package application

import (
	"context"
	"testing"
	"time"

	"lambda_create_post/internal/domain"
)

type postRepositoryStub struct {
	post domain.PostModel
	err  error
}

func (s *postRepositoryStub) SavePost(_ context.Context, post domain.PostModel) error {
	s.post = post
	return s.err
}

func TestCreatePostCreatesSparseDraftModel(t *testing.T) {
	repository := &postRepositoryStub{}
	useCase := NewCreatePostUseCase(repository)
	useCase.now = func() time.Time { return time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC) }
	useCase.newID = func() (string, error) { return "post-123", nil }

	post, err := useCase.CreatePost(context.Background(), " author-1 ", " First draft ", domain.PostStatusDraft)
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if post.PostID != "post-123" || post.AuthorID != "author-1" || post.Description != "First draft" {
		t.Fatalf("CreatePost() post = %+v", post)
	}
	if post.PublishedAt != "" || post.Status != domain.PostStatusDraft {
		t.Fatalf("draft post = %+v; expected no publishedAt and DRAFT status", post)
	}
	if repository.post != post {
		t.Fatalf("saved post = %+v, want %+v", repository.post, post)
	}
}

func TestCreatePostSetsPublishedAtForPublishedPost(t *testing.T) {
	repository := &postRepositoryStub{}
	useCase := NewCreatePostUseCase(repository)
	useCase.now = func() time.Time { return time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC) }
	useCase.newID = func() (string, error) { return "post-123", nil }

	post, err := useCase.CreatePost(context.Background(), "author-1", "Ready", domain.PostStatusPublished)
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if post.PublishedAt == "" || post.PublishedAt != post.CreatedAt {
		t.Fatalf("published post = %+v; expected publishedAt equal to createdAt", post)
	}
}
