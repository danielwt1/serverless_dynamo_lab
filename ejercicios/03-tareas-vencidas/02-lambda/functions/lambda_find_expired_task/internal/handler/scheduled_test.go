package handler

import (
	"context"
	"errors"
	"testing"
	"time"
)

type findExpiredTasksStub struct {
	executionTime string
	err           error
}

func (stub *findExpiredTasksStub) Execute(_ context.Context, executionTime string) error {
	stub.executionTime = executionTime
	return stub.err
}

func TestScheduledHandlerUsesExplicitExecutionTime(t *testing.T) {
	useCase := &findExpiredTasksStub{}
	handler := NewScheduledHandler(useCase)
	if err := handler.Handle(context.Background(), ScheduledEvent{ExecutionTime: " 2026-09-14T05:00:00Z "}); err != nil {
		t.Fatal(err)
	}
	if useCase.executionTime != "2026-09-14T05:00:00Z" {
		t.Fatalf("execution_time = %q", useCase.executionTime)
	}
}

func TestScheduledHandlerFallsBackToEventBridgeTime(t *testing.T) {
	useCase := &findExpiredTasksStub{}
	handler := NewScheduledHandler(useCase)
	eventTime := time.Date(2026, 9, 14, 0, 0, 0, 123, time.FixedZone("COT", -5*60*60))
	if err := handler.Handle(context.Background(), ScheduledEvent{Time: eventTime}); err != nil {
		t.Fatal(err)
	}
	if useCase.executionTime != "2026-09-14T05:00:00.000000123Z" {
		t.Fatalf("execution_time = %q", useCase.executionTime)
	}
}

func TestScheduledHandlerValidatesEventAndWrapsUseCaseError(t *testing.T) {
	handler := NewScheduledHandler(&findExpiredTasksStub{})
	if err := handler.Handle(context.Background(), ScheduledEvent{}); err == nil {
		t.Fatal("se esperaba error para un evento sin fecha")
	}

	wantErr := errors.New("DynamoDB unavailable")
	useCase := &findExpiredTasksStub{err: wantErr}
	err := NewScheduledHandler(useCase).Handle(context.Background(), ScheduledEvent{ExecutionTime: "2026-09-14T05:00:00Z"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, quiere %v", err, wantErr)
	}
}
