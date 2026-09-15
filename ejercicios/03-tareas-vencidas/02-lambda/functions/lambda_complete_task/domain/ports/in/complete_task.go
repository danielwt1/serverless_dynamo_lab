package in

import (
	"context"
	"lambda_complete_task/domain"
)

type CompleteTask interface {
	Execute(ctx context.Context, ownerID, taskID string) (domain.Task, error)
}
