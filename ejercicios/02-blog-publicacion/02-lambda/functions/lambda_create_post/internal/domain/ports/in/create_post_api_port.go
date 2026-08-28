package in

import (
	"context"
	"lambda_create_post/internal/domain"
)

type CreatePostAPIPort interface {
	CreatePost(ctx context.Context, authorID, description string, status domain.PostStatus) (domain.PostModel, error)
}
