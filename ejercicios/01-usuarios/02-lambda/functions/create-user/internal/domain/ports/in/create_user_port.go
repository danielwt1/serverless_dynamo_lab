package in

import (
	"context"
	"create-user/internal/domain"
)

type CreateUserApiPort interface {
	CreateUser(ctx context.Context, user domain.User) error
}
