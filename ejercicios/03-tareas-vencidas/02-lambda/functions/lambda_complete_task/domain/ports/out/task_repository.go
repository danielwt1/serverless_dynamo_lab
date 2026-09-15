package out

import (
	"context"
	"lambda_complete_task/domain"
	"time"
)

type TaskRepository interface {
	CompletePendingTask(ctx context.Context, ownerID, taskID string, completedAt time.Time) (domain.Task, error)
}
