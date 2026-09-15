package ports

import (
	"context"
	"lambda_task_overdue_consumer/domain"
	"time"
)

type ConsumeTaskOverdue interface {
	Execute(ctx context.Context, event domain.TaskOverdueEvent) (bool, error)
}

type NotificationRepository interface {
	Claim(ctx context.Context, eventID string, consumedAt time.Time) (bool, error)
}
