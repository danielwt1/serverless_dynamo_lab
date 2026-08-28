package application

import (
	"context"
	"create-user/internal/domain"
	"create-user/internal/domain/ports/out"
)

type CreateUserUsecase struct {
	userRepo out.CreateUserPort
}

func NewCreateUserUsecase(port out.CreateUserPort) *CreateUserUsecase {
	return &CreateUserUsecase{
		userRepo: port,
	}
}
func (u *CreateUserUsecase) CreateUser(ctx context.Context, user domain.User) error {
	return u.userRepo.Create(ctx, user)
}
