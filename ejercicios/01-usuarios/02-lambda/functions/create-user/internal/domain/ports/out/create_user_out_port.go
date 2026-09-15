package out

import (
	"context"
	"create-user/internal/domain"
)

type CreateUserPort interface {
	Create(ctx context.Context, user domain.User) error
}
