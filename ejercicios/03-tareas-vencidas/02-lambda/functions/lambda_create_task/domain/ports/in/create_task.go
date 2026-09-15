package in

import (
	"context"
	"lambda_create_task/domain"
	"time"
)

type CreateTask interface {
	Execute(ctx context.Context, ownerID, description string, expiredAt time.Time) (domain.Task, error)
}
