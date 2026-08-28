package application

import (
	"context"
	"lambda_posts_mutable/internal/domain"
	"lambda_posts_mutable/internal/domain/ports/out"
)

type GetInfoUseCase struct {
	repository out.GetInfoPort
}

func NewGetInfoUseCase(repository out.GetInfoPort) *GetInfoUseCase {
	return &GetInfoUseCase{repository: repository}
}
func (rp *GetInfoUseCase) FindPublishedPosts(ctx context.Context, author_id string, cursor string, limit int32) (domain.PostsPage, error) {
	return rp.repository.GetPublishedPosts(ctx, author_id, "PUBLISHED", cursor, limit)

}

func (rp *GetInfoUseCase) FindDraftPosts(ctx context.Context, authorID string, cursor string, limit int32) (domain.PostsPage, error) {
	return rp.repository.GetDraftPosts(ctx, authorID, cursor, limit)
}
