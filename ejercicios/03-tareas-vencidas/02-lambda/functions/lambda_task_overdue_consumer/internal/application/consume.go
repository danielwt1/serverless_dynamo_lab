package application

import (
	"context"
	"errors"
	"fmt"
	"lambda_task_overdue_consumer/domain"
	"lambda_task_overdue_consumer/domain/ports"
	"time"
)

type ConsumeTaskOverdueUseCase struct {
	repository ports.NotificationRepository
	now        func() time.Time
}

func NewConsumeTaskOverdueUseCase(repository ports.NotificationRepository) *ConsumeTaskOverdueUseCase {
	return &ConsumeTaskOverdueUseCase{repository: repository, now: time.Now}
}

func (useCase *ConsumeTaskOverdueUseCase) Execute(ctx context.Context, event domain.TaskOverdueEvent) (bool, error) {
	if useCase.repository == nil {
		return false, errors.New("el repositorio de notificaciones es obligatorio")
	}
	if event.EventType != "TaskOverdue" || event.EventID == "" || event.TaskID == "" || event.OwnerID == "" || event.ExpiredAt.IsZero() {
		return false, errors.New("evento TaskOverdue inválido")
	}
	claimed, err := useCase.repository.Claim(ctx, event.EventID, useCase.now().UTC())
	if err != nil {
		return false, fmt.Errorf("reservar consumo %s: %w", event.EventID, err)
	}
	return claimed, nil
}

var _ ports.ConsumeTaskOverdue = (*ConsumeTaskOverdueUseCase)(nil)
