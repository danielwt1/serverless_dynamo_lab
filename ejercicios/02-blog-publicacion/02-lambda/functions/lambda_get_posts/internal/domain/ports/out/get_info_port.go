package out

import (
	"context"
	"lambda_posts_mutable/internal/domain"
)

type GetInfoPort interface {
	GetPublishedPosts(ctx context.Context, author_id string, status string, cursor string, limit int32) (domain.PostsPage, error)

	GetDraftPosts(ctx context.Context, authorID string, cursor string, limit int32) (domain.PostsPage, error)
}
