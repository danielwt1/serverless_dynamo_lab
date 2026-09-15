package application

import (
	"context"
	"errors"
	"lambda_create_task/domain"
	"testing"
	"time"
)

type taskRepositoryStub struct {
	task domain.Task
	err  error
}

func (stub *taskRepositoryStub) CreatePendingTask(_ context.Context, task domain.Task) error {
	stub.task = task
	return stub.err
}

func TestCreateTaskBuildsPendingTask(t *testing.T) {
	now := time.Date(2026, 9, 14, 5, 0, 0, 0, time.UTC)
	repository := &taskRepositoryStub{}
	useCase := NewCreateTaskUseCase(repository)
	useCase.now = func() time.Time { return now }
	useCase.newID = func() (string, error) { return "task-1", nil }

	task, err := useCase.Execute(context.Background(), " owner-1 ", " Pagar factura ", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if task.TaskID != "task-1" || task.OwnerID != "owner-1" || task.Description != "Pagar factura" || task.Status != domain.PendingStatus || len(task.ShardID) != 2 {
		t.Fatalf("tarea incorrecta: %+v", task)
	}
	if repository.task != task {
		t.Fatalf("tarea no persistida: %+v", repository.task)
	}
}

func TestCreateTaskRejectsExpiredDateAndPreservesRepositoryError(t *testing.T) {
	now := time.Date(2026, 9, 14, 5, 0, 0, 0, time.UTC)
	useCase := NewCreateTaskUseCase(&taskRepositoryStub{})
	useCase.now = func() time.Time { return now }
	if _, err := useCase.Execute(context.Background(), "owner-1", "task", now); !errors.Is(err, domain.ErrInvalidTask) {
		t.Fatalf("error=%v", err)
	}

	wantErr := errors.New("DynamoDB unavailable")
	useCase = NewCreateTaskUseCase(&taskRepositoryStub{err: wantErr})
	useCase.now = func() time.Time { return now }
	useCase.newID = func() (string, error) { return "task-1", nil }
	if _, err := useCase.Execute(context.Background(), "owner-1", "task", now.Add(time.Hour)); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, quiere %v", err, wantErr)
	}
}
