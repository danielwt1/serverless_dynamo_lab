package out

import (
	"context"
	"lambda_create_post/internal/domain"
)

type CreatePostPort interface {
	SavePost(ctx context.Context, post domain.PostModel) error
}
