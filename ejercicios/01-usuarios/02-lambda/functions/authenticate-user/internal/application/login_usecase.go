package application

import (
	"context"
	"errors"

	"authenticate-user/internal/domain"
	"authenticate-user/internal/domain/ports/out"
)

type UserLogin struct {
	persistencePort out.UserPersistencePort
}

func NewUserLogin(persistencePort out.UserPersistencePort) *UserLogin {
	return &UserLogin{
		persistencePort: persistencePort,
	}
}
func (ul *UserLogin) GetUserByEmail(ctx context.Context, email string, password string) (string, error) {
	user, err := ul.persistencePort.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.UserNotFound) {
			return "", domain.UserNotFoundOrPassWordWrong
		}
		return "", err
	}
	if err := validatePCredentials(email, password, user); err != nil {
		return "", err
	}

	return user.UserId, nil
}

func validatePCredentials(email string, password string, user domain.User) error {
	if password != user.Password || user.Email != email {
		return domain.UserNotFoundOrPassWordWrong
	}
	return nil
}
