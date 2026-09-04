package in

import (
	"context"
	"lambda_posts_mutable/internal/domain"
)

type GetInfoApiPort interface {
	FindPublishedPosts(ctx context.Context, author_id string, cursor string, limit int32) (domain.PostsPage, error)

	FindDraftPosts(ctx context.Context, authorID string, cursor string, limit int32) (domain.PostsPage, error)
}
