package out

import (
	"context"

	"list-active-users/internal/domain"
)

type ActiveUsersPersistencePort interface {
	ListActiveUsers(ctx context.Context, limit int32, cursor string) ([]domain.User, string, error)
}
