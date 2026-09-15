package application

import (
	"context"
	"errors"
	"fmt"
	"lambda_find_expired_task/domain"
	portin "lambda_find_expired_task/domain/ports/in"
	"lambda_find_expired_task/domain/ports/out"
	"sync"
	"time"
)

const (
	defaultShardCount    = 6
	defaultCheckpointTTL = 7 * 24 * time.Hour
)

type FindExpiredTasksUseCase struct {
	tasks         out.TaskOverDueRepository
	processes     out.ProcessRepository[domain.ProcessModel]
	shards        out.ProcessRepository[domain.ProcessShardModel]
	progress      out.ProcessProgressRepository
	recipient     out.TaskOverdueRecipient
	shardCount    int
	checkpointTTL time.Duration
	now           func() time.Time
}

func NewFindExpiredTasksUseCase(
	tasks out.TaskOverDueRepository,
	processes out.ProcessRepository[domain.ProcessModel],
	shards out.ProcessRepository[domain.ProcessShardModel],
	progress out.ProcessProgressRepository,
	recipient out.TaskOverdueRecipient,
) *FindExpiredTasksUseCase {
	return &FindExpiredTasksUseCase{
		tasks:         tasks,
		processes:     processes,
		shards:        shards,
		progress:      progress,
		recipient:     recipient,
		shardCount:    defaultShardCount,
		checkpointTTL: defaultCheckpointTTL,
		now:           time.Now,
	}
}

// Execute recupera un batch abierto o crea uno nuevo. Primero deja los eventos
// y checkpoints en DynamoDB; después publica el outbox y cierra el batch.
func (uc *FindExpiredTasksUseCase) Execute(ctx context.Context, executionTime string) error {
	if err := uc.validateDependencies(); err != nil {
		return err
	}
	triggerAt, err := parseExecutionTime(executionTime)
	if err != nil {
		return err
	}

	process, newCheckpoints, completed, err := uc.resolveProcess(ctx, triggerAt)
	if err != nil || completed {
		return err
	}
	runID := process.ActiveRunId
	executionAt, err := parseExecutionTime(process.ExecutionTime)
	if err != nil {
		return fmt.Errorf("execution_time persistido inválido: %w", err)
	}

	if process.Status != "PUBLISHING" {
		checkpoints := newCheckpoints
		if checkpoints == nil {
			checkpoints, err = uc.loadCheckpoints(ctx, runID)
			if err != nil {
				return uc.failProcess(ctx, runID, err)
			}
		}
		if err := uc.processShards(ctx, runID, executionAt, checkpoints); err != nil {
			return uc.failProcess(ctx, runID, err)
		}
		if err := uc.progress.SetProcessStatus(ctx, runID, "PUBLISHING", "", uc.timestamp()); err != nil {
			return uc.failProcess(ctx, runID, fmt.Errorf("preparar publicación: %w", err))
		}
	}

	if err := uc.publishOutbox(ctx, runID); err != nil {
		return uc.failProcess(ctx, runID, err)
	}
	if err := uc.progress.SetProcessStatus(ctx, runID, "COMPLETED", "", uc.timestamp()); err != nil {
		return fmt.Errorf("completar proceso: %w", err)
	}
	return nil
}

func (uc *FindExpiredTasksUseCase) resolveProcess(ctx context.Context, triggerAt time.Time) (domain.ProcessModel, []domain.ProcessShardModel, bool, error) {
	process, err := uc.processes.FindOpenProcess(ctx)
	if err == nil {
		process, err = normalizedOpenProcess(process)
		return process, nil, false, err
	}
	if !errors.Is(err, out.ErrProcessNotFound) {
		return domain.ProcessModel{}, nil, false, fmt.Errorf("buscar proceso abierto: %w", err)
	}

	canonicalExecutionTime := triggerAt.Format(time.RFC3339Nano)
	runID := "OVERDUE_RUN#" + canonicalExecutionTime
	process, err = uc.processes.FindProcess(ctx, map[string]string{"id": runID, "sk": "META"})
	if err == nil {
		if process.Status == "COMPLETED" {
			return process, nil, true, nil
		}
		process, err = normalizedOpenProcess(process)
		return process, nil, false, err
	}
	if !errors.Is(err, out.ErrProcessNotFound) {
		return domain.ProcessModel{}, nil, false, fmt.Errorf("buscar proceso del trigger: %w", err)
	}

	now := uc.timestamp()
	process = domain.ProcessModel{
		Status:        "RUNNING",
		ExecutionTime: canonicalExecutionTime,
		ActiveRunId:   runID,
		UpdatedAt:     now,
	}
	checkpoints := make([]domain.ProcessShardModel, uc.shardCount)
	ttl := uc.now().UTC().Add(uc.checkpointTTL).Unix()
	for index := range checkpoints {
		checkpoints[index] = domain.ProcessShardModel{
			Status:     "PENDING",
			RunId:      runID,
			ShardId:    fmt.Sprintf("%02d", index),
			Updated_at: now,
			Ttl:        ttl,
		}
	}
	if err := uc.processes.InitProcessAndShardProcess(ctx, process, checkpoints); err != nil {
		return domain.ProcessModel{}, nil, false, fmt.Errorf("inicializar proceso: %w", err)
	}
	return process, checkpoints, false, nil
}

func normalizedOpenProcess(process domain.ProcessModel) (domain.ProcessModel, error) {
	switch process.Status {
	case "RUNNING", "PUBLISHING", "FAILED_RETRYABLE":
	default:
		return domain.ProcessModel{}, fmt.Errorf("estado de proceso no soportado: %q", process.Status)
	}
	if process.ExecutionTime == "" {
		return domain.ProcessModel{}, errors.New("el proceso abierto no tiene execution_time")
	}
	if process.ActiveRunId == "" {
		process.ActiveRunId = "OVERDUE_RUN#" + process.ExecutionTime
	}
	return process, nil
}

func (uc *FindExpiredTasksUseCase) loadCheckpoints(ctx context.Context, runID string) ([]domain.ProcessShardModel, error) {
	checkpoints := make([]domain.ProcessShardModel, 0, uc.shardCount)
	for index := 0; index < uc.shardCount; index++ {
		shardID := fmt.Sprintf("%02d", index)
		checkpoint, err := uc.shards.FindProcess(ctx, map[string]string{
			"id": runID,
			"sk": "SHARD#" + shardID,
		})
		if err != nil {
			return nil, fmt.Errorf("buscar checkpoint del shard %s: %w", shardID, err)
		}
		if checkpoint.ShardId == "" {
			checkpoint.ShardId = shardID
		}
		if checkpoint.RunId == "" {
			checkpoint.RunId = runID
		}
		checkpoints = append(checkpoints, checkpoint)
	}
	return checkpoints, nil
}

func (uc *FindExpiredTasksUseCase) processShards(ctx context.Context, runID string, executionAt time.Time, checkpoints []domain.ProcessShardModel) error {
	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	errorsByShard := make(chan error, len(checkpoints))
	var workers sync.WaitGroup
	for _, checkpoint := range checkpoints {
		checkpoint := checkpoint
		if checkpoint.Status == "COMPLETED" {
			continue
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := uc.processShard(workerContext, runID, executionAt, checkpoint); err != nil {
				errorsByShard <- fmt.Errorf("procesar shard %s: %w", checkpoint.ShardId, err)
				cancel()
			}
		}()
	}
	workers.Wait()
	close(errorsByShard)
	var workerErrors []error
	for workerErr := range errorsByShard {
		workerErrors = append(workerErrors, workerErr)
	}
	return errors.Join(workerErrors...)
}

func (uc *FindExpiredTasksUseCase) processShard(ctx context.Context, runID string, executionAt time.Time, checkpoint domain.ProcessShardModel) error {
	switch checkpoint.Status {
	case "PENDING", "IN_PROGRESS":
	case "COMPLETED":
		return nil
	default:
		return fmt.Errorf("estado de checkpoint no soportado: %q", checkpoint.Status)
	}
	for {
		page, err := uc.tasks.GetOverDueTasks(ctx, checkpoint.Cursor, checkpoint.ShardId, executionAt.Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("consultar tareas: %w", err)
		}
		for _, task := range page.Task {
			event, err := overdueEvent(task, executionAt)
			if err != nil {
				return err
			}
			if err := uc.progress.ReserveTaskOverdue(ctx, runID, event); err != nil {
				return fmt.Errorf("reservar evento de la tarea %s: %w", task.TaskId, err)
			}
		}
		if page.Cursor != "" && page.Cursor == checkpoint.Cursor {
			return errors.New("DynamoDB devolvió el mismo cursor de continuación")
		}
		checkpoint.Cursor = page.Cursor
		checkpoint.Pages_completed++
		checkpoint.Updated_at = uc.timestamp()
		checkpoint.Status = "IN_PROGRESS"
		if page.Cursor == "" {
			checkpoint.Status = "COMPLETED"
		}
		if err := uc.progress.SaveShardCheckpoint(ctx, runID, checkpoint); err != nil {
			return fmt.Errorf("guardar checkpoint: %w", err)
		}
		if page.Cursor == "" {
			return nil
		}
	}
}

func (uc *FindExpiredTasksUseCase) publishOutbox(ctx context.Context, runID string) error {
	cursor := ""
	for {
		events, nextCursor, err := uc.progress.ListReadyTaskOverdue(ctx, runID, cursor)
		if err != nil {
			return fmt.Errorf("leer eventos pendientes: %w", err)
		}
		for _, event := range events {
			if err := uc.recipient.PublishTaskOverdue(ctx, event); err != nil {
				return fmt.Errorf("publicar tarea %s: %w", event.TaskId, err)
			}
			if err := uc.progress.MarkTaskOverduePublished(ctx, runID, event, uc.timestamp()); err != nil {
				return fmt.Errorf("confirmar publicación de la tarea %s: %w", event.TaskId, err)
			}
		}
		if nextCursor == "" {
			return nil
		}
		if nextCursor == cursor {
			return errors.New("DynamoDB devolvió el mismo cursor del outbox")
		}
		cursor = nextCursor
	}
}

func overdueEvent(task domain.TaskModel, executionAt time.Time) (domain.TaskOverdueEvent, error) {
	if task.TaskId == "" {
		return domain.TaskOverdueEvent{}, errors.New("la tarea vencida no tiene task_id")
	}
	if task.Expire_at == nil {
		return domain.TaskOverdueEvent{}, fmt.Errorf("la tarea %s no tiene expired_at", task.TaskId)
	}
	expiredAt := task.Expire_at.UTC()
	return domain.TaskOverdueEvent{
		EventId:       task.TaskId + "#" + expiredAt.Format(time.RFC3339Nano),
		TaskId:        task.TaskId,
		OwnerId:       task.OwnerId,
		Description:   task.Description,
		ExpiredAt:     expiredAt,
		ExecutionTime: executionAt,
	}, nil
}

func parseExecutionTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("execution_time debe usar RFC3339: %w", err)
	}
	return parsed.UTC(), nil
}

func (uc *FindExpiredTasksUseCase) failProcess(ctx context.Context, runID string, processErr error) error {
	updateErr := uc.progress.SetProcessStatus(context.WithoutCancel(ctx), runID, "FAILED_RETRYABLE", processErr.Error(), uc.timestamp())
	if updateErr != nil {
		return errors.Join(processErr, fmt.Errorf("marcar proceso fallido: %w", updateErr))
	}
	return processErr
}

func (uc *FindExpiredTasksUseCase) validateDependencies() error {
	if uc.tasks == nil || uc.processes == nil || uc.shards == nil || uc.progress == nil || uc.recipient == nil {
		return errors.New("el caso de uso requiere todos sus puertos")
	}
	if uc.shardCount < 1 || uc.shardCount > 99 {
		return errors.New("shardCount debe estar entre 1 y 99")
	}
	return nil
}

func (uc *FindExpiredTasksUseCase) timestamp() string {
	return uc.now().UTC().Format(time.RFC3339Nano)
}

var _ portin.FindExpiredTasks = (*FindExpiredTasksUseCase)(nil)
