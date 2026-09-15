package out

import (
	"context"
	"lambda_create_task/domain"
)

type TaskRepository interface {
	CreatePendingTask(ctx context.Context, task domain.Task) error
}
