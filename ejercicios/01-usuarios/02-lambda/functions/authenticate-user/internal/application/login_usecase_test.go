package application

import (
	"context"
	"testing"

	"authenticate-user/internal/domain"
)

// userPersistenceStub es un doble de prueba: sustituye DynamoDB en este test.
// Conserva el email recibido para que también podamos comprobar la colaboración.
type userPersistenceStub struct {
	user          domain.User
	err           error
	receivedEmail string
}

func (s *userPersistenceStub) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	s.receivedEmail = email
	return s.user, s.err
}

func TestUserLogin_GetUserByEmail_ReturnsUserIDWhenCredentialsAreValid(t *testing.T) {
	// Arrange: preparamos una dependencia controlada, sin AWS ni DynamoDB.
	repository := &userPersistenceStub{
		user: domain.User{
			Email:    "ana@example.com",
			Password: "secret",
			UserId:   "user-123",
		},
	}
	useCase := NewUserLogin(repository)

	// Act: ejecutamos el comportamiento que queremos probar.
	userID, err := useCase.GetUserByEmail(context.Background(), "ana@example.com", "secret")

	// Assert: verificamos el resultado y la llamada al puerto.
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v, want nil", err)
	}
	if userID != "user-123" {
		t.Errorf("GetUserByEmail() userID = %q, want %q", userID, "user-123")
	}
	if repository.receivedEmail != "ana@example.com" {
		t.Errorf("repository received email = %q, want %q", repository.receivedEmail, "ana@example.com")
	}
}
