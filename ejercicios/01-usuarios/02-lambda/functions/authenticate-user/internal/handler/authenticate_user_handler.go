package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"authenticate-user/internal/domain"
	"authenticate-user/internal/domain/ports/in"

	"github.com/aws/aws-lambda-go/events"
)

type AuthenticateUserHandler struct {
	userLogin in.UserLoginAPIPort
}

type loginRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

type loginResponse struct {
	Message string `json:"message"`
	UserID  string `json:"userId,omitempty"`
}

func NewAuthenticateUserHandler(userLogin in.UserLoginAPIPort) *AuthenticateUserHandler {
	return &AuthenticateUserHandler{userLogin: userLogin}
}

func (h *AuthenticateUserHandler) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var body loginRequest
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return response(400, "invalid request body", ""), nil
	}

	if body.Email == nil || strings.TrimSpace(*body.Email) == "" {
		return response(400, "email is required", ""), nil
	}
	if body.Password == nil || strings.TrimSpace(*body.Password) == "" {
		return response(400, "password is required", ""), nil
	}

	userID, err := h.userLogin.GetUserByEmail(ctx, strings.TrimSpace(*body.Email), *body.Password)
	if err != nil {
		if errors.Is(err, domain.UserNotFoundOrPassWordWrong) || errors.Is(err, domain.UserNotFound) {
			return response(404, err.Error(), ""), nil
		}
		return response(500, "internal server error", ""), nil
	}

	return response(200, "user authenticated successfully", userID), nil
}

func response(statusCode int, message, userID string) events.APIGatewayV2HTTPResponse {
	// loginResponse has only string fields, so json.Marshal cannot return an error.
	body, _ := json.Marshal(loginResponse{Message: message, UserID: userID})

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}
}
