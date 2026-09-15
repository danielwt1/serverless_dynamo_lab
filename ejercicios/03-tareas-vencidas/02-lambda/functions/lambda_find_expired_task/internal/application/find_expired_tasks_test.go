package application

import (
	"context"
	"errors"
	"lambda_find_expired_task/domain"
	"lambda_find_expired_task/domain/ports/out"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

type processRepositoryStub[V any] struct {
	find     func(map[string]string) (V, error)
	findOpen func() (domain.ProcessModel, error)
	init     func(domain.ProcessModel, []domain.ProcessShardModel) error
}

func (s *processRepositoryStub[V]) FindProcess(_ context.Context, keys map[string]string) (V, error) {
	return s.find(keys)
}

func (s *processRepositoryStub[V]) FindOpenProcess(context.Context) (domain.ProcessModel, error) {
	if s.findOpen == nil {
		return domain.ProcessModel{}, out.ErrProcessNotFound
	}
	return s.findOpen()
}

func (s *processRepositoryStub[V]) InitProcessAndShardProcess(_ context.Context, process domain.ProcessModel, shards []domain.ProcessShardModel) error {
	if s.init == nil {
		return nil
	}
	return s.init(process, shards)
}

type taskRepositoryStub struct {
	mu    sync.Mutex
	calls []taskCall
	find  func(cursor, shardID, executionTime string) (domain.QueryResult, error)
}

type taskCall struct {
	cursor        string
	shardID       string
	executionTime string
}

func (s *taskRepositoryStub) GetOverDueTasks(_ context.Context, cursor, shardID, executionTime string) (domain.QueryResult, error) {
	s.mu.Lock()
	s.calls = append(s.calls, taskCall{cursor: cursor, shardID: shardID, executionTime: executionTime})
	s.mu.Unlock()
	return s.find(cursor, shardID, executionTime)
}

type progressRepositoryStub struct {
	mu          sync.Mutex
	checkpoints []domain.ProcessShardModel
	reserved    []domain.TaskOverdueEvent
	published   []domain.TaskOverdueEvent
	statuses    []string
	lastErrors  []string
}

func (s *progressRepositoryStub) ReserveTaskOverdue(_ context.Context, _ string, event domain.TaskOverdueEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reserved = append(s.reserved, event)
	return nil
}

func (s *progressRepositoryStub) SaveShardCheckpoint(_ context.Context, _ string, shard domain.ProcessShardModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoints = append(s.checkpoints, shard)
	return nil
}

func (s *progressRepositoryStub) ListReadyTaskOverdue(context.Context, string, string) ([]domain.TaskOverdueEvent, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	published := make(map[string]bool, len(s.published))
	for _, event := range s.published {
		published[event.EventId] = true
	}
	ready := make([]domain.TaskOverdueEvent, 0, len(s.reserved))
	for _, event := range s.reserved {
		if !published[event.EventId] {
			ready = append(ready, event)
		}
	}
	return ready, "", nil
}

func (s *progressRepositoryStub) MarkTaskOverduePublished(_ context.Context, _ string, event domain.TaskOverdueEvent, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published = append(s.published, event)
	return nil
}

func (s *progressRepositoryStub) SetProcessStatus(_ context.Context, _, status, lastError, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses = append(s.statuses, status)
	s.lastErrors = append(s.lastErrors, lastError)
	return nil
}

type recipientStub struct {
	mu     sync.Mutex
	events []domain.TaskOverdueEvent
	err    error
}

func (s *recipientStub) PublishTaskOverdue(_ context.Context, event domain.TaskOverdueEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return s.err
}

func TestFindExpiredTasksUseCaseInitializesAndProcessesAllShards(t *testing.T) {
	fixedNow := time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)
	expiredAt := time.Date(2026, 9, 13, 10, 30, 0, 0, time.UTC)
	var initializedProcess domain.ProcessModel
	var initializedShards []domain.ProcessShardModel
	processes := &processRepositoryStub[domain.ProcessModel]{
		findOpen: func() (domain.ProcessModel, error) {
			return domain.ProcessModel{}, out.ErrProcessNotFound
		},
		find: func(map[string]string) (domain.ProcessModel, error) {
			return domain.ProcessModel{}, out.ErrProcessNotFound
		},
		init: func(process domain.ProcessModel, shards []domain.ProcessShardModel) error {
			if len(shards) != 6 {
				t.Fatalf("el caso de uso intentó inicializar %d shards", len(shards))
			}
			initializedProcess = process
			initializedShards = append([]domain.ProcessShardModel(nil), shards...)
			return nil
		},
	}
	shards := &processRepositoryStub[domain.ProcessShardModel]{
		find: func(map[string]string) (domain.ProcessShardModel, error) {
			t.Fatal("no debe leer checkpoints recién creados")
			return domain.ProcessShardModel{}, nil
		},
	}
	tasks := &taskRepositoryStub{find: func(_, shardID, _ string) (domain.QueryResult, error) {
		if shardID != "02" {
			return domain.QueryResult{Task: []domain.TaskModel{}}, nil
		}
		return domain.QueryResult{Task: []domain.TaskModel{{
			TaskId:      "task-1",
			OwnerId:     "owner-1",
			Description: "pagar factura",
			Expire_at:   &expiredAt,
			Status:      "PENDING",
		}}}, nil
	}}
	progress := &progressRepositoryStub{}
	recipient := &recipientStub{}
	useCase := NewFindExpiredTasksUseCase(tasks, processes, shards, progress, recipient)
	useCase.now = func() time.Time { return fixedNow }

	if err := useCase.Execute(context.Background(), "2026-09-14T15:00:00-05:00"); err != nil {
		t.Fatal(err)
	}

	if initializedProcess.ExecutionTime != "2026-09-14T20:00:00Z" || initializedProcess.ActiveRunId != "OVERDUE_RUN#2026-09-14T20:00:00Z" {
		t.Fatalf("identidad de proceso incorrecta: %+v", initializedProcess)
	}
	if len(initializedShards) != 6 {
		t.Fatalf("shards inicializados = %d, quiere 6", len(initializedShards))
	}
	for index, shard := range initializedShards {
		if shard.ShardId != []string{"00", "01", "02", "03", "04", "05"}[index] || shard.Status != "PENDING" {
			t.Fatalf("checkpoint inicial incorrecto: %+v", shard)
		}
	}
	if len(recipient.events) != 1 {
		t.Fatalf("eventos publicados = %d, quiere 1", len(recipient.events))
	}
	if len(progress.reserved) != 1 || len(progress.published) != 1 {
		t.Fatalf("outbox incorrecto: reserved=%d published=%d", len(progress.reserved), len(progress.published))
	}
	event := recipient.events[0]
	if event.EventId != "task-1#2026-09-13T10:30:00Z" || event.ExecutionTime.Format(time.RFC3339) != "2026-09-14T20:00:00Z" {
		t.Fatalf("evento incorrecto: %+v", event)
	}
	if len(progress.checkpoints) != 6 {
		t.Fatalf("checkpoints confirmados = %d, quiere 6", len(progress.checkpoints))
	}
	if !reflect.DeepEqual(progress.statuses, []string{"PUBLISHING", "COMPLETED"}) {
		t.Fatalf("estados = %v", progress.statuses)
	}
}

func TestFindExpiredTasksUseCaseResumesOnlyIncompleteShards(t *testing.T) {
	processes := &processRepositoryStub[domain.ProcessModel]{findOpen: func() (domain.ProcessModel, error) {
		return domain.ProcessModel{
			Status:        "FAILED_RETRYABLE",
			ExecutionTime: "2026-09-14T20:00:00Z",
			ActiveRunId:   "OVERDUE_RUN#2026-09-14T20:00:00Z",
		}, nil
	}}
	shards := &processRepositoryStub[domain.ProcessShardModel]{find: func(keys map[string]string) (domain.ProcessShardModel, error) {
		if keys["sk"] == "SHARD#00" {
			return domain.ProcessShardModel{ShardId: "00", Status: "COMPLETED"}, nil
		}
		return domain.ProcessShardModel{ShardId: "01", Status: "IN_PROGRESS", Cursor: "saved", Pages_completed: 3}, nil
	}}
	tasks := &taskRepositoryStub{find: func(cursor, _, executionTime string) (domain.QueryResult, error) {
		if executionTime != "2026-09-14T20:00:00Z" {
			t.Fatalf("execution_time = %q", executionTime)
		}
		if cursor == "saved" {
			return domain.QueryResult{Task: []domain.TaskModel{}, Cursor: "next"}, nil
		}
		return domain.QueryResult{Task: []domain.TaskModel{}}, nil
	}}
	progress := &progressRepositoryStub{}
	useCase := NewFindExpiredTasksUseCase(tasks, processes, shards, progress, &recipientStub{})
	useCase.shardCount = 2
	useCase.now = func() time.Time { return time.Date(2026, 9, 14, 20, 1, 0, 0, time.UTC) }

	if err := useCase.Execute(context.Background(), "2026-09-14T20:00:00Z"); err != nil {
		t.Fatal(err)
	}

	if len(tasks.calls) != 2 || tasks.calls[0].shardID != "01" || tasks.calls[0].cursor != "saved" || tasks.calls[1].cursor != "next" {
		t.Fatalf("consultas = %+v", tasks.calls)
	}
	if len(progress.checkpoints) != 2 || progress.checkpoints[0].Pages_completed != 4 || progress.checkpoints[1].Pages_completed != 5 {
		t.Fatalf("progreso = %+v", progress.checkpoints)
	}
	statuses := []string{progress.checkpoints[0].Status, progress.checkpoints[1].Status}
	sort.Strings(statuses)
	if !reflect.DeepEqual(statuses, []string{"COMPLETED", "IN_PROGRESS"}) {
		t.Fatalf("estados de checkpoint = %v", statuses)
	}
	if !reflect.DeepEqual(progress.statuses, []string{"PUBLISHING", "COMPLETED"}) {
		t.Fatalf("estados del proceso = %v", progress.statuses)
	}
}

func TestFindExpiredTasksUseCaseKeepsOutboxReadyOnPublishError(t *testing.T) {
	expiredAt := time.Date(2026, 9, 13, 10, 30, 0, 0, time.UTC)
	processes := &processRepositoryStub[domain.ProcessModel]{
		findOpen: func() (domain.ProcessModel, error) { return domain.ProcessModel{}, out.ErrProcessNotFound },
		find: func(map[string]string) (domain.ProcessModel, error) {
			return domain.ProcessModel{}, out.ErrProcessNotFound
		},
	}
	shards := &processRepositoryStub[domain.ProcessShardModel]{find: func(map[string]string) (domain.ProcessShardModel, error) {
		return domain.ProcessShardModel{}, errors.New("unexpected")
	}}
	tasks := &taskRepositoryStub{find: func(_, _, _ string) (domain.QueryResult, error) {
		return domain.QueryResult{Task: []domain.TaskModel{{TaskId: "task-1", Expire_at: &expiredAt}}}, nil
	}}
	progress := &progressRepositoryStub{}
	wantErr := errors.New("SNS unavailable")
	useCase := NewFindExpiredTasksUseCase(tasks, processes, shards, progress, &recipientStub{err: wantErr})
	useCase.shardCount = 1
	useCase.now = func() time.Time { return time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC) }

	err := useCase.Execute(context.Background(), "2026-09-14T20:00:00Z")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, quiere %v", err, wantErr)
	}
	if len(progress.checkpoints) != 1 || len(progress.reserved) != 1 || len(progress.published) != 0 {
		t.Fatalf("estado del outbox incorrecto: checkpoints=%d reserved=%d published=%d", len(progress.checkpoints), len(progress.reserved), len(progress.published))
	}
	if !reflect.DeepEqual(progress.statuses, []string{"PUBLISHING", "FAILED_RETRYABLE"}) || len(progress.lastErrors) != 2 || progress.lastErrors[1] == "" {
		t.Fatalf("fallo no persistido: statuses=%v errors=%v", progress.statuses, progress.lastErrors)
	}
}

func TestFindExpiredTasksUseCaseRejectsInvalidExecutionTime(t *testing.T) {
	useCase := NewFindExpiredTasksUseCase(
		&taskRepositoryStub{},
		&processRepositoryStub[domain.ProcessModel]{},
		&processRepositoryStub[domain.ProcessShardModel]{},
		&progressRepositoryStub{},
		&recipientStub{},
	)
	if err := useCase.Execute(context.Background(), "14-09-2026"); err == nil {
		t.Fatal("se esperaba error")
	}
}
