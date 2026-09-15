package application

import (
	"context"
	"errors"
	"get_task_lambda/internal/domain"
	"testing"
)

type taskQueryRepositoryStub struct {
	result domain.QueryResult
	err    error
	method string
	owner  string
	cursor string
}

func (stub *taskQueryRepositoryStub) GetUserTasks(_ context.Context, owner, cursor string) (domain.QueryResult, error) {
	stub.method, stub.owner, stub.cursor = "all", owner, cursor
	return stub.result, stub.err
}

func (stub *taskQueryRepositoryStub) GetPendingTasksByUser(_ context.Context, owner, cursor string) (domain.QueryResult, error) {
	stub.method, stub.owner, stub.cursor = "pending", owner, cursor
	return stub.result, stub.err
}

func TestTasksQueryApiDelegatesQueries(t *testing.T) {
	want := domain.QueryResult{Task: []domain.TaskModel{{TaskId: "task-1"}}, Cursor: "next"}
	stub := &taskQueryRepositoryStub{result: want}
	useCase := NewTasksQueryApi(stub)

	got, err := useCase.FindTasks(context.Background(), "owner-1", "cursor")
	if err != nil || len(got.Task) != 1 || stub.method != "all" || stub.owner != "owner-1" || stub.cursor != "cursor" {
		t.Fatalf("resultado=%+v error=%v llamada=%+v", got, err, stub)
	}
	got, err = useCase.FindPendingTasksByUser(context.Background(), "owner-1", "")
	if err != nil || len(got.Task) != 1 || stub.method != "pending" {
		t.Fatalf("resultado=%+v error=%v llamada=%+v", got, err, stub)
	}
}

func TestTasksQueryApiPreservesRepositoryError(t *testing.T) {
	wantErr := errors.New("DynamoDB unavailable")
	useCase := NewTasksQueryApi(&taskQueryRepositoryStub{err: wantErr})
	if _, err := useCase.FindTasks(context.Background(), "owner-1", ""); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, quiere %v", err, wantErr)
	}
}

func TestTasksQueryApiRejectsEmptyOwner(t *testing.T) {
	if _, err := NewTasksQueryApi(&taskQueryRepositoryStub{}).FindTasks(context.Background(), " ", ""); err == nil {
		t.Fatal("se esperaba error")
	}
}
