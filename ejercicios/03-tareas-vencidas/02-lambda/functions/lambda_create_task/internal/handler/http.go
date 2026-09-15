package handler

import (
	"context"
	"encoding/json"
	"errors"
	"lambda_create_task/domain"
	portin "lambda_create_task/domain/ports/in"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type createTaskRequest struct {
	Description string    `json:"description"`
	ExpiredAt   time.Time `json:"expired_at"`
}

type HTTPHandler struct{ useCase portin.CreateTask }

func NewHTTPHandler(useCase portin.CreateTask) *HTTPHandler { return &HTTPHandler{useCase: useCase} }

func (handler *HTTPHandler) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if handler.useCase == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "servicio no configurado"})
	}
	ownerID := strings.TrimSpace(request.PathParameters["ownerId"])
	var input createTaskRequest
	decoder := json.NewDecoder(strings.NewReader(request.Body))
	decoder.DisallowUnknownFields()
	if ownerID == "" || decoder.Decode(&input) != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"message": "ownerId y body válido son obligatorios"})
	}
	task, err := handler.useCase.Execute(ctx, ownerID, input.Description, input.ExpiredAt)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTask):
			return jsonResponse(http.StatusBadRequest, map[string]string{"message": err.Error()})
		case errors.Is(err, domain.ErrTaskExists):
			return jsonResponse(http.StatusConflict, map[string]string{"message": err.Error()})
		default:
			return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "no se pudo crear la tarea"})
		}
	}
	return jsonResponse(http.StatusCreated, task)
}

func jsonResponse(status int, value any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json; charset=utf-8"},
		Body:       string(body),
	}, nil
}
