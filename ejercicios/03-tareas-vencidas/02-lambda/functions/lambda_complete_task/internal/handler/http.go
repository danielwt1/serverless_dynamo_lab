package handler

import (
	"context"
	"encoding/json"
	"errors"
	"lambda_complete_task/domain"
	portin "lambda_complete_task/domain/ports/in"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type HTTPHandler struct{ useCase portin.CompleteTask }

func NewHTTPHandler(useCase portin.CompleteTask) *HTTPHandler { return &HTTPHandler{useCase: useCase} }

func (handler *HTTPHandler) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if handler.useCase == nil {
		return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "servicio no configurado"})
	}
	ownerID := strings.TrimSpace(request.PathParameters["ownerId"])
	taskID := strings.TrimSpace(request.PathParameters["taskId"])
	task, err := handler.useCase.Execute(ctx, ownerID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidID):
			return jsonResponse(http.StatusBadRequest, map[string]string{"message": err.Error()})
		case errors.Is(err, domain.ErrTaskNotFound):
			return jsonResponse(http.StatusNotFound, map[string]string{"message": err.Error()})
		case errors.Is(err, domain.ErrTaskNotPending):
			return jsonResponse(http.StatusConflict, map[string]string{"message": err.Error()})
		default:
			return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "no se pudo completar la tarea"})
		}
	}
	return jsonResponse(http.StatusOK, task)
}

func jsonResponse(status int, value any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: status, Headers: map[string]string{"Content-Type": "application/json; charset=utf-8"}, Body: string(body)}, nil
}
