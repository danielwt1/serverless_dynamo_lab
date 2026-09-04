package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"list-active-users/internal/domain"
	"list-active-users/internal/domain/ports/in"

	"github.com/aws/aws-lambda-go/events"
)

const defaultLimit int32 = 10

type ListActiveUsersHandler struct {
	listActiveUsers in.ListActiveUsersAPIPort
}

type listActiveUsersResponse struct {
	Items      []domain.User `json:"items"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

type errorResponse struct {
	Message string `json:"message"`
}

func NewListActiveUsersHandler(listActiveUsers in.ListActiveUsersAPIPort) *ListActiveUsersHandler {
	return &ListActiveUsersHandler{listActiveUsers: listActiveUsers}
}

func (h *ListActiveUsersHandler) List(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	limit, err := parseLimit(event.QueryStringParameters["limit"])
	if err != nil {
		return errorJSON(400, err.Error())
	}

	users, nextCursor, err := h.listActiveUsers.ListActiveUsers(ctx, limit, event.QueryStringParameters["cursor"])
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCursor) {
			return errorJSON(400, err.Error())
		}
		return errorJSON(500, "could not list active users")
	}

	return jsonResponse(200, listActiveUsersResponse{Items: users, NextCursor: nextCursor})
}

func parseLimit(rawLimit string) (int32, error) {
	if rawLimit == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.ParseInt(rawLimit, 10, 32)
	if err != nil || limit < 1 || limit > 100 {
		return 0, domain.ErrInvalidLimit
	}
	return int32(limit), nil
}

func errorJSON(statusCode int, message string) (events.APIGatewayV2HTTPResponse, error) {
	return jsonResponse(statusCode, errorResponse{Message: message})
}

func jsonResponse(statusCode int, responseBody any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(responseBody)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}
