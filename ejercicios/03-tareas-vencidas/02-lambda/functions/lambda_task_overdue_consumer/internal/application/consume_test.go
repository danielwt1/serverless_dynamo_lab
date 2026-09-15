package application

import (
	"context"
	"lambda_task_overdue_consumer/domain"
	"testing"
	"time"
)

type notificationRepositoryStub struct {
	eventID string
	claimed bool
}

func (stub *notificationRepositoryStub) Claim(_ context.Context, eventID string, _ time.Time) (bool, error) {
	stub.eventID = eventID
	return stub.claimed, nil
}

func TestConsumeTaskOverdueClaimsEvent(t *testing.T) {
	repository := &notificationRepositoryStub{claimed: true}
	useCase := NewConsumeTaskOverdueUseCase(repository)
	event := domain.TaskOverdueEvent{EventType: "TaskOverdue", EventID: "event-1", TaskID: "task-1", OwnerID: "owner-1", ExpiredAt: time.Now()}
	claimed, err := useCase.Execute(context.Background(), event)
	if err != nil || !claimed || repository.eventID != "event-1" {
		t.Fatalf("claimed=%v error=%v eventID=%q", claimed, err, repository.eventID)
	}
}

func TestConsumeTaskOverdueRejectsInvalidEvent(t *testing.T) {
	if _, err := NewConsumeTaskOverdueUseCase(&notificationRepositoryStub{}).Execute(context.Background(), domain.TaskOverdueEvent{}); err == nil {
		t.Fatal("se esperaba error")
	}
}
