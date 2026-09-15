package infraestructure

import (
	"context"
	"encoding/json"
	"errors"
	"lambda_find_expired_task/domain"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type snsPublisherStub struct {
	input *sns.PublishInput
	err   error
}

func (stub *snsPublisherStub) Publish(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
	stub.input = input
	return &sns.PublishOutput{}, stub.err
}

func TestSNSRecipientPublishesTaskOverdueEnvelope(t *testing.T) {
	client := &snsPublisherStub{}
	recipient := NewSNSRecipient(client, "arn:aws:sns:us-east-1:123456789012:task-overdue")
	event := validTaskOverdueEvent()

	if err := recipient.PublishTaskOverdue(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if client.input == nil || *client.input.TopicArn != "arn:aws:sns:us-east-1:123456789012:task-overdue" {
		t.Fatalf("publicación incorrecta: %+v", client.input)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(*client.input.Message), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["event_type"] != "TaskOverdue" || payload["event_id"] != event.EventId || payload["task_id"] != event.TaskId {
		t.Fatalf("payload incorrecto: %v", payload)
	}
	if *client.input.MessageAttributes["event_id"].StringValue != event.EventId {
		t.Fatalf("atributos SNS incorrectos: %v", client.input.MessageAttributes)
	}
}

func TestSNSRecipientConfiguresFIFOAndPreservesPublishError(t *testing.T) {
	wantErr := errors.New("SNS unavailable")
	client := &snsPublisherStub{err: wantErr}
	recipient := NewSNSRecipient(client, "arn:aws:sns:us-east-1:123456789012:task-overdue.fifo")

	err := recipient.PublishTaskOverdue(context.Background(), validTaskOverdueEvent())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, quiere %v", err, wantErr)
	}
	if client.input.MessageGroupId == nil || client.input.MessageDeduplicationId == nil {
		t.Fatalf("faltan parámetros FIFO: %+v", client.input)
	}
}

func validTaskOverdueEvent() domain.TaskOverdueEvent {
	return domain.TaskOverdueEvent{
		EventId:       "task-1#2026-09-13T10:30:00Z",
		TaskId:        "task-1",
		OwnerId:       "owner-1",
		Description:   "pagar factura",
		ExpiredAt:     time.Date(2026, 9, 13, 10, 30, 0, 0, time.UTC),
		ExecutionTime: time.Date(2026, 9, 14, 5, 0, 0, 0, time.UTC),
	}
}
