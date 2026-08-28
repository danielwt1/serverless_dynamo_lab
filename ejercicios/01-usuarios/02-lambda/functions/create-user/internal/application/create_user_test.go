package application

import (
	"context"
	"errors"
	"testing"

	"create-user/internal/domain"
)

type createUserStub struct {
	user domain.User
	err  error
}

func (s *createUserStub) Create(_ context.Context, user domain.User) error {
	s.user = user
	return s.err
}

func TestCreateUserUsecase_CreateUser(t *testing.T) {
	want := domain.User{Name: "Ana", LastName: "Perez", Email: "ana@example.com"}
	repository := &createUserStub{}

	err := NewCreateUserUsecase(repository).CreateUser(context.Background(), want)

	if err != nil {
		t.Fatalf("CreateUser() error = %v, want nil", err)
	}
	if repository.user != want {
		t.Errorf("Create() received %#v, want %#v", repository.user, want)
	}
}

func TestCreateUserUsecase_CreateUser_ReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("DynamoDB unavailable")
	err := NewCreateUserUsecase(&createUserStub{err: wantErr}).CreateUser(context.Background(), domain.User{})
	if !errors.Is(err, wantErr) {
		t.Errorf("CreateUser() error = %v, want %v", err, wantErr)
	}
}
