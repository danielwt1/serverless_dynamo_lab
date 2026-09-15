package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"lambda_create_task/domain"
	portin "lambda_create_task/domain/ports/in"
	"lambda_create_task/domain/ports/out"
	"strings"
	"time"
)

const shardCount = 6

type CreateTaskUseCase struct {
	repository out.TaskRepository
	now        func() time.Time
	newID      func() (string, error)
}

func NewCreateTaskUseCase(repository out.TaskRepository) *CreateTaskUseCase {
	return &CreateTaskUseCase{repository: repository, now: time.Now, newID: newUUID}
}

func (useCase *CreateTaskUseCase) Execute(ctx context.Context, ownerID, description string, expiredAt time.Time) (domain.Task, error) {
	ownerID = strings.TrimSpace(ownerID)
	description = strings.TrimSpace(description)
	if useCase.repository == nil {
		return domain.Task{}, errors.New("el repositorio de tareas es obligatorio")
	}
	if ownerID == "" || description == "" || len(description) > 1000 || expiredAt.IsZero() {
		return domain.Task{}, domain.ErrInvalidTask
	}
	now := useCase.now().UTC()
	expiredAt = expiredAt.UTC()
	if !expiredAt.After(now) {
		return domain.Task{}, fmt.Errorf("%w: expired_at debe estar en el futuro", domain.ErrInvalidTask)
	}
	taskID, err := useCase.newID()
	if err != nil {
		return domain.Task{}, fmt.Errorf("generar task_id: %w", err)
	}
	task := domain.Task{
		TaskID:      taskID,
		OwnerID:     ownerID,
		Description: description,
		CreatedAt:   now,
		ExpiredAt:   expiredAt,
		Status:      domain.PendingStatus,
		ShardID:     shardFor(taskID),
	}
	if err := useCase.repository.CreatePendingTask(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("crear tarea: %w", err)
	}
	return task, nil
}

func shardFor(taskID string) string {
	hash := sha256.Sum256([]byte(taskID))
	return fmt.Sprintf("%02d", int(hash[0])%shardCount)
}

func newUUID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

var _ portin.CreateTask = (*CreateTaskUseCase)(nil)
