package out

import (
	"context"
	"errors"
	"lambda_find_expired_task/domain"
)

var ErrProcessNotFound = errors.New("proceso no encontrado")

type TaskOverDueRepository interface {
	GetOverDueTasks(ctx context.Context, cursor, shardId, executionTime string) (domain.QueryResult, error)
}

type ProcessRepository[V any] interface {
	FindProcess(context context.Context, keys map[string]string) (V, error)
	FindOpenProcess(ctx context.Context) (domain.ProcessModel, error)
	InitProcessAndShardProcess(ctx context.Context, process domain.ProcessModel, shards []domain.ProcessShardModel) error
}

type ProcessProgressRepository interface {
	ReserveTaskOverdue(ctx context.Context, runId string, event domain.TaskOverdueEvent) error
	SaveShardCheckpoint(ctx context.Context, runId string, shard domain.ProcessShardModel) error
	ListReadyTaskOverdue(ctx context.Context, runId, cursor string) ([]domain.TaskOverdueEvent, string, error)
	MarkTaskOverduePublished(ctx context.Context, runId string, event domain.TaskOverdueEvent, publishedAt string) error
	SetProcessStatus(ctx context.Context, runId, status, lastError, updatedAt string) error
}
