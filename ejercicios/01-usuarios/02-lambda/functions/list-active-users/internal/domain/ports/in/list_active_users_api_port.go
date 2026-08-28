package in

import (
	"context"

	"list-active-users/internal/domain"
)

type ListActiveUsersAPIPort interface {
	ListActiveUsers(ctx context.Context, limit int32, cursor string) ([]domain.User, string, error)
}
