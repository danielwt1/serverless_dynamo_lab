package in

import (
	"context"
	"get_task_lambda/internal/domain"
)

type TasksQueryApi interface {
	FindTasks(context context.Context, userId string, cursor string) (domain.QueryResult, error)
	FindPendingTasksByUser(context context.Context, userId string, cursor string) (domain.QueryResult, error)
}
