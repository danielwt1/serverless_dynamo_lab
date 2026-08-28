package handler

import (
	"context"
	"errors"

	"create-user/internal/domain"
	"create-user/internal/domain/ports/in"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
)

type UserRequest struct {
	Name        string `json:"name"`
	LastName    string `json:"lastName"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DateOfBirth string `json:"dateOfBirth"`
}

type UserResponse struct {
	Message string `json:"message"`
}

type CreateUserHandler struct {
	createUserPort in.CreateUserApiPort
}

func NewCreateUserHandler(port in.CreateUserApiPort) *CreateUserHandler {
	return &CreateUserHandler{
		createUserPort: port,
	}
}

func (h *CreateUserHandler) CreateUser(
	ctx context.Context,
	event events.APIGatewayV2HTTPRequest,
) (events.APIGatewayV2HTTPResponse, error) {
	requestBody := UserRequest{}
	if err := json.Unmarshal([]byte(event.Body), &requestBody); err != nil {
		return response(400, "invalid JSON")
	}

	user := domain.User{
		Name:        requestBody.Name,
		LastName:    requestBody.LastName,
		Email:       requestBody.Email,
		Password:    requestBody.Password,
		DateOfBirth: requestBody.DateOfBirth,
	}
	if err := h.createUserPort.CreateUser(ctx, user); err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return response(409, "user already exists")
		}

		return response(500, "could not create user")
	}

	return response(201, "User created successfully")
}

func response(statusCode int, message string) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(UserResponse{Message: message})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Body:       string(body),
	}, nil
}
