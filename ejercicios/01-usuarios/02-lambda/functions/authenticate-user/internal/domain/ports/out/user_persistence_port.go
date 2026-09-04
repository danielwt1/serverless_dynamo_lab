package out

import (
	"authenticate-user/internal/domain"
	"context"
)

type UserPersistencePort interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}
