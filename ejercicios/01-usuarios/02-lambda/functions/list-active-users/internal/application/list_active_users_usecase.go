package application

import (
	"context"

	"list-active-users/internal/domain"
	"list-active-users/internal/domain/ports/out"
)

type ListActiveUsersUseCase struct {
	persistencePort out.ActiveUsersPersistencePort
}

func NewListActiveUsersUseCase(persistencePort out.ActiveUsersPersistencePort) *ListActiveUsersUseCase {
	return &ListActiveUsersUseCase{persistencePort: persistencePort}
}

func (u *ListActiveUsersUseCase) ListActiveUsers(ctx context.Context, limit int32, cursor string) ([]domain.User, string, error) {
	return u.persistencePort.ListActiveUsers(ctx, limit, cursor)
}
