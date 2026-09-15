package handler

import (
	"context"
	"errors"
	"fmt"
	portin "lambda_find_expired_task/domain/ports/in"
	"strings"
	"time"
)

type ScheduledEvent struct {
	ExecutionTime string    `json:"execution_time"`
	Time          time.Time `json:"time"`
}

type ScheduledHandler struct {
	useCase portin.FindExpiredTasks
}

func NewScheduledHandler(useCase portin.FindExpiredTasks) *ScheduledHandler {
	return &ScheduledHandler{useCase: useCase}
}

func (handler *ScheduledHandler) Handle(ctx context.Context, event ScheduledEvent) error {
	if handler.useCase == nil {
		return errors.New("el caso de uso FindExpiredTasks es obligatorio")
	}
	executionTime := strings.TrimSpace(event.ExecutionTime)
	if executionTime == "" && !event.Time.IsZero() {
		executionTime = event.Time.UTC().Format(time.RFC3339Nano)
	}
	if executionTime == "" {
		return errors.New("el evento debe incluir execution_time o time")
	}
	if err := handler.useCase.Execute(ctx, executionTime); err != nil {
		return fmt.Errorf("procesar tareas vencidas para %s: %w", executionTime, err)
	}
	return nil
}
