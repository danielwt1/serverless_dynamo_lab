package application

import (
	"context"
	"errors"
	"fmt"
	"get_task_lambda/internal/domain"
	"get_task_lambda/internal/domain/in"
	"get_task_lambda/internal/domain/out"
	"strings"
)

type TasksQueryApi struct {
	repository out.TaskQueryRepository
}

func NewTasksQueryApi(repository out.TaskQueryRepository) *TasksQueryApi {
	return &TasksQueryApi{
		repository: repository,
	}
}
func (ft *TasksQueryApi) FindTasks(context context.Context, userId string, cursor string) (domain.QueryResult, error) {
	if ft.repository == nil {
		return domain.QueryResult{}, errors.New("el repositorio de tareas es obligatorio")
	}
	if strings.TrimSpace(userId) == "" {
		return domain.QueryResult{}, errors.New("ownerId es obligatorio")
	}
	resultQuery, err := ft.repository.GetUserTasks(context, userId, cursor)
	if err != nil {
		return domain.QueryResult{}, fmt.Errorf("consultar tareas: %w", err)
	}
	return resultQuery, nil
}
func (ft *TasksQueryApi) FindPendingTasksByUser(ctx context.Context, userId string, cursor string) (domain.QueryResult, error) {
	if ft.repository == nil {
		return domain.QueryResult{}, errors.New("el repositorio de tareas es obligatorio")
	}
	if strings.TrimSpace(userId) == "" {
		return domain.QueryResult{}, errors.New("ownerId es obligatorio")
	}
	resultQuery, err := ft.repository.GetPendingTasksByUser(ctx, userId, cursor)
	if err != nil {
		return domain.QueryResult{}, fmt.Errorf("consultar tareas pendientes: %w", err)
	}
	return resultQuery, nil
}

var _ in.TasksQueryApi = (*TasksQueryApi)(nil)
