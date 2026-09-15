package application

import (
	"context"
	"errors"
	"fmt"
	"lambda_complete_task/domain"
	portin "lambda_complete_task/domain/ports/in"
	"lambda_complete_task/domain/ports/out"
	"strings"
	"time"
)

type CompleteTaskUseCase struct {
	repository out.TaskRepository
	now        func() time.Time
}

func NewCompleteTaskUseCase(repository out.TaskRepository) *CompleteTaskUseCase {
	return &CompleteTaskUseCase{repository: repository, now: time.Now}
}

func (useCase *CompleteTaskUseCase) Execute(ctx context.Context, ownerID, taskID string) (domain.Task, error) {
	if useCase.repository == nil {
		return domain.Task{}, errors.New("el repositorio de tareas es obligatorio")
	}
	ownerID = strings.TrimSpace(ownerID)
	taskID = strings.TrimSpace(taskID)
	if ownerID == "" || taskID == "" {
		return domain.Task{}, domain.ErrInvalidID
	}
	task, err := useCase.repository.CompletePendingTask(ctx, ownerID, taskID, useCase.now().UTC())
	if err != nil {
		return domain.Task{}, fmt.Errorf("completar tarea: %w", err)
	}
	return task, nil
}

var _ portin.CompleteTask = (*CompleteTaskUseCase)(nil)
