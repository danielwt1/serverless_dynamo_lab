package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"create-user/internal/domain"
	"github.com/aws/aws-lambda-go/events"
)

type createUserHandlerStub struct {
	user domain.User
	err  error
}

func (s *createUserHandlerStub) CreateUser(_ context.Context, user domain.User) error {
	s.user = user
	return s.err
}

func TestCreateUserHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name, body, message string
		err                  error
		status               int
	}{
		{name: "rejects invalid JSON", body: "{", status: 400, message: "invalid JSON"},
		{name: "creates user", body: `{"name":"Ana","lastName":"Perez","email":"ana@example.com","password":"secret","dateOfBirth":"2000-01-02"}`, status: 201, message: "User created successfully"},
		{name: "reports duplicate user", body: `{}`, err: domain.ErrUserAlreadyExists, status: 409, message: "user already exists"},
		{name: "hides unexpected errors", body: `{}`, err: errors.New("timeout"), status: 500, message: "could not create user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &createUserHandlerStub{err: tt.err}
			response, err := NewCreateUserHandler(service).CreateUser(context.Background(), events.APIGatewayV2HTTPRequest{Body: tt.body})
			if err != nil { t.Fatalf("CreateUser() error = %v", err) }
			if response.StatusCode != tt.status { t.Errorf("status = %d, want %d", response.StatusCode, tt.status) }
			var payload UserResponse
			if err := json.Unmarshal([]byte(response.Body), &payload); err != nil { t.Fatalf("invalid response JSON: %v", err) }
			if payload.Message != tt.message { t.Errorf("message = %q, want %q", payload.Message, tt.message) }
			if tt.name == "creates user" && service.user.Email != "ana@example.com" { t.Errorf("received user = %#v", service.user) }
		})
	}
}
