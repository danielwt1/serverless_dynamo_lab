package infraestructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lambda_find_expired_task/domain"
	"lambda_find_expired_task/domain/ports/out"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type SNSPublisher interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type SNSRecipient struct {
	client   SNSPublisher
	topicARN string
}

func NewSNSRecipient(client SNSPublisher, topicARN string) *SNSRecipient {
	return &SNSRecipient{client: client, topicARN: strings.TrimSpace(topicARN)}
}

func (recipient *SNSRecipient) PublishTaskOverdue(ctx context.Context, event domain.TaskOverdueEvent) error {
	if recipient.client == nil {
		return errors.New("el cliente SNS es obligatorio")
	}
	if recipient.topicARN == "" {
		return errors.New("TASK_OVERDUE_TOPIC_ARN es obligatorio")
	}
	if event.EventId == "" || event.TaskId == "" || event.ExpiredAt.IsZero() || event.ExecutionTime.IsZero() {
		return errors.New("event_id, task_id, expired_at y execution_time son obligatorios")
	}

	payload, err := json.Marshal(struct {
		EventType string `json:"event_type"`
		domain.TaskOverdueEvent
	}{EventType: "TaskOverdue", TaskOverdueEvent: event})
	if err != nil {
		return fmt.Errorf("serializar TaskOverdue: %w", err)
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(recipient.topicARN),
		Message:  aws.String(string(payload)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("TaskOverdue"),
			},
			"event_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.EventId),
			},
		},
	}
	if strings.HasSuffix(recipient.topicARN, ".fifo") {
		input.MessageGroupId = aws.String("TASK_OVERDUE")
		input.MessageDeduplicationId = aws.String(event.EventId)
	}
	if _, err := recipient.client.Publish(ctx, input); err != nil {
		return fmt.Errorf("publicar TaskOverdue %s: %w", event.EventId, err)
	}
	return nil
}

var _ out.TaskOverdueRecipient = (*SNSRecipient)(nil)
