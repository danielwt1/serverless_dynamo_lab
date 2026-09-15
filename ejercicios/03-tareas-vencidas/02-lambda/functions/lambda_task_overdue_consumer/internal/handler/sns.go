package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"lambda_task_overdue_consumer/domain"
	"lambda_task_overdue_consumer/domain/ports"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
)

type SNSHandler struct{ useCase ports.ConsumeTaskOverdue }

func NewSNSHandler(useCase ports.ConsumeTaskOverdue) *SNSHandler {
	return &SNSHandler{useCase: useCase}
}

func (handler *SNSHandler) Handle(ctx context.Context, snsEvent events.SNSEvent) error {
	if handler.useCase == nil {
		return fmt.Errorf("el caso de uso es obligatorio")
	}
	for _, record := range snsEvent.Records {
		var event domain.TaskOverdueEvent
		if err := json.Unmarshal([]byte(record.SNS.Message), &event); err != nil {
			return fmt.Errorf("decodificar mensaje SNS %s: %w", record.SNS.MessageID, err)
		}
		claimed, err := handler.useCase.Execute(ctx, event)
		if err != nil {
			return fmt.Errorf("consumir mensaje SNS %s: %w", record.SNS.MessageID, err)
		}
		if claimed {
			slog.InfoContext(ctx, "task overdue notification consumed", "event_id", event.EventID, "task_id", event.TaskID, "owner_id", event.OwnerID)
		}
	}
	return nil
}
