package application

import (
	"context"
	"testing"

	"lambda_posts_mutable/internal/domain"
)

type getInfoRepositoryStub struct {
	publishedAuthor, publishedStatus, publishedCursor string
	publishedLimit                                  int32
	draftAuthor, draftCursor                        string
	draftLimit                                      int32
}

func (s *getInfoRepositoryStub) GetPublishedPosts(_ context.Context, authorID, status, cursor string, limit int32) (domain.PostsPage, error) {
	s.publishedAuthor, s.publishedStatus, s.publishedCursor, s.publishedLimit = authorID, status, cursor, limit
	return domain.PostsPage{NextCursor: "published-next"}, nil
}

func (s *getInfoRepositoryStub) GetDraftPosts(_ context.Context, authorID, cursor string, limit int32) (domain.PostsPage, error) {
	s.draftAuthor, s.draftCursor, s.draftLimit = authorID, cursor, limit
	return domain.PostsPage{NextCursor: "draft-next"}, nil
}

func TestGetInfoUseCase_FindPublishedPosts_UsesPublishedStatus(t *testing.T) {
	repository := &getInfoRepositoryStub{}
	page, err := NewGetInfoUseCase(repository).FindPublishedPosts(context.Background(), "author-1", "cursor-1", 20)
	if err != nil || page.NextCursor != "published-next" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	if repository.publishedAuthor != "author-1" || repository.publishedStatus != "PUBLISHED" || repository.publishedCursor != "cursor-1" || repository.publishedLimit != 20 {
		t.Errorf("repository = %+v", repository)
	}
}

func TestGetInfoUseCase_FindDraftPosts_ForwardsRequest(t *testing.T) {
	repository := &getInfoRepositoryStub{}
	page, err := NewGetInfoUseCase(repository).FindDraftPosts(context.Background(), "author-1", "cursor-1", 20)
	if err != nil || page.NextCursor != "draft-next" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	if repository.draftAuthor != "author-1" || repository.draftCursor != "cursor-1" || repository.draftLimit != 20 {
		t.Errorf("repository = %+v", repository)
	}
}
