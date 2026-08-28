package in

import (
	"context"
)

type UserLoginAPIPort interface {
	GetUserByEmail(ctx context.Context, email string, password string) (string, error)
}
