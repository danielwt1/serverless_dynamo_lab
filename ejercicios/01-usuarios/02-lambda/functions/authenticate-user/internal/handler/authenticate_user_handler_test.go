package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"authenticate-user/internal/domain"

	"github.com/aws/aws-lambda-go/events"
)

type userLoginStub struct {
	userID           string
	err              error
	receivedEmail    string
	receivedPassword string
}

func (s *userLoginStub) GetUserByEmail(_ context.Context, email, password string) (string, error) {
	s.receivedEmail = email
	s.receivedPassword = password
	return s.userID, s.err
}

func TestAuthenticateUserHandler_Handle(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		userID      string
		loginErr    error
		wantStatus  int
		wantMessage string
		wantUserID  string
		wantEmail   string
		wantPass    string
	}{
		{name: "rejects invalid JSON", body: "{", wantStatus: 400, wantMessage: "invalid request body"},
		{name: "requires email", body: `{"password":"secret"}`, wantStatus: 400, wantMessage: "email is required"},
		{name: "rejects blank email", body: `{"email":"  ","password":"secret"}`, wantStatus: 400, wantMessage: "email is required"},
		{name: "requires password", body: `{"email":"ana@example.com"}`, wantStatus: 400, wantMessage: "password is required"},
		{name: "rejects blank password", body: `{"email":"ana@example.com","password":" "}`, wantStatus: 400, wantMessage: "password is required"},
		{name: "authenticates a valid request", body: `{"email":" ana@example.com ","password":"secret"}`, userID: "user-123", wantStatus: 200, wantMessage: "user authenticated successfully", wantUserID: "user-123", wantEmail: "ana@example.com", wantPass: "secret"},
		{name: "maps invalid credentials to not found", body: `{"email":"ana@example.com","password":"bad"}`, loginErr: domain.UserNotFoundOrPassWordWrong, wantStatus: 404, wantMessage: "user not found or password wrong", wantEmail: "ana@example.com", wantPass: "bad"},
		{name: "maps a missing user to not found", body: `{"email":"ana@example.com","password":"secret"}`, loginErr: domain.UserNotFound, wantStatus: 404, wantMessage: "user not found", wantEmail: "ana@example.com", wantPass: "secret"},
		{name: "hides unexpected errors", body: `{"email":"ana@example.com","password":"secret"}`, loginErr: errors.New("database timeout"), wantStatus: 500, wantMessage: "internal server error", wantEmail: "ana@example.com", wantPass: "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login := &userLoginStub{userID: tt.userID, err: tt.loginErr}
			h := NewAuthenticateUserHandler(login)

			response, err := h.Handle(context.Background(), events.APIGatewayV2HTTPRequest{Body: tt.body})

			if err != nil {
				t.Fatalf("Handle() error = %v, want nil", err)
			}
			if response.StatusCode != tt.wantStatus {
				t.Errorf("Handle() status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if response.Headers["Content-Type"] != "application/json" {
				t.Errorf("Handle() Content-Type = %q, want application/json", response.Headers["Content-Type"])
			}
			var body loginResponse
			if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
				t.Fatalf("response body is not valid JSON: %v", err)
			}
			if body.Message != tt.wantMessage || body.UserID != tt.wantUserID {
				t.Errorf("response body = %#v, want message=%q userID=%q", body, tt.wantMessage, tt.wantUserID)
			}
			if login.receivedEmail != tt.wantEmail || login.receivedPassword != tt.wantPass {
				t.Errorf("login received email=%q password=%q, want email=%q password=%q", login.receivedEmail, login.receivedPassword, tt.wantEmail, tt.wantPass)
			}
		})
	}
}
