package out

import (
	"context"
	"get_task_lambda/internal/domain"
)

type TaskQueryRepository interface {
	GetUserTasks(ctx context.Context, userId string, cursor string) (domain.QueryResult, error)
	GetPendingTasksByUser(ctx context.Context, userId string, cursor string) (domain.QueryResult, error)
}
