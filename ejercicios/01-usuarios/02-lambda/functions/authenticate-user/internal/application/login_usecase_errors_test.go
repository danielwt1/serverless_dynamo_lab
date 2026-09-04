package application

import (
	"context"
	"errors"
	"testing"

	"authenticate-user/internal/domain"
)

func TestUserLogin_GetUserByEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		user     domain.User
		repoErr  error
		wantID   string
		wantErr  error
	}{
		{
			name:    "translates a missing user into invalid credentials",
			email:   "ana@example.com",
			repoErr: domain.UserNotFound,
			wantErr: domain.UserNotFoundOrPassWordWrong,
		},
		{
			name:     "rejects an incorrect password",
			email:    "ana@example.com",
			password: "wrong-password",
			user:     domain.User{Email: "ana@example.com", Password: "secret", UserId: "user-123"},
			wantErr:  domain.UserNotFoundOrPassWordWrong,
		},
		{
			name:     "rejects a user returned for another email",
			email:    "ana@example.com",
			password: "secret",
			user:     domain.User{Email: "other@example.com", Password: "secret", UserId: "user-123"},
			wantErr:  domain.UserNotFoundOrPassWordWrong,
		},
		{
			name:    "propagates an unexpected repository error",
			email:   "ana@example.com",
			repoErr: errors.New("DynamoDB timeout"),
			wantErr: errors.New("DynamoDB timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &userPersistenceStub{user: tt.user, err: tt.repoErr}
			useCase := NewUserLogin(repository)

			userID, err := useCase.GetUserByEmail(context.Background(), tt.email, tt.password)

			if userID != tt.wantID {
				t.Errorf("GetUserByEmail() userID = %q, want %q", userID, tt.wantID)
			}
			if tt.repoErr != nil && !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
				t.Errorf("GetUserByEmail() error = %v, want %v", err, tt.wantErr)
				return
			}
			if tt.repoErr == nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("GetUserByEmail() error = %v, want %v", err, tt.wantErr)
			}
			if repository.receivedEmail != tt.email {
				t.Errorf("repository received email = %q, want %q", repository.receivedEmail, tt.email)
			}
		})
	}
}
