package infraestructure

import (
	"context"
	"errors"
	"fmt"
	"lambda_task_overdue_consumer/domain/ports"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoNotificationClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type DynamoNotificationRepository struct {
	client    DynamoNotificationClient
	tableName string
}

func NewDynamoNotificationRepository(client DynamoNotificationClient, tableName string) *DynamoNotificationRepository {
	return &DynamoNotificationRepository{client: client, tableName: tableName}
}

func (repository *DynamoNotificationRepository) Claim(ctx context.Context, eventID string, consumedAt time.Time) (bool, error) {
	if repository.client == nil || repository.tableName == "" || eventID == "" {
		return false, errors.New("cliente, tabla y event_id son obligatorios")
	}
	key := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "TASK_OVERDUE#" + eventID},
		"SK": &types.AttributeValueMemberS{Value: "EVENT#TASK_OVERDUE"},
	}
	item, err := repository.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(repository.tableName), ConsistentRead: aws.Bool(true), Key: key,
	})
	if err != nil {
		return false, fmt.Errorf("leer marker: %w", err)
	}
	if len(item.Item) == 0 {
		return false, errors.New("marker de notificación no encontrado")
	}
	status, _ := item.Item["status"].(*types.AttributeValueMemberS)
	if status == nil || status.Value != "PUBLISHED" {
		return false, errors.New("la publicación aún no fue confirmada")
	}
	if _, consumed := item.Item["consumed_at"]; consumed {
		return false, nil
	}
	_, err = repository.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           aws.String(repository.tableName),
		Key:                 key,
		ConditionExpression: aws.String("#status = :published AND attribute_not_exists(consumed_at)"),
		UpdateExpression:    aws.String("SET consumed_at = :consumed_at, consumer = :consumer"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":published":   &types.AttributeValueMemberS{Value: "PUBLISHED"},
			":consumed_at": &types.AttributeValueMemberS{Value: consumedAt.UTC().Format(time.RFC3339Nano)},
			":consumer":    &types.AttributeValueMemberS{Value: "TASK_OVERDUE_LOG"},
		},
	})
	if err != nil {
		var conditional *types.ConditionalCheckFailedException
		if errors.As(err, &conditional) {
			return false, nil
		}
		return false, fmt.Errorf("actualizar marker: %w", err)
	}
	return true, nil
}

var _ ports.NotificationRepository = (*DynamoNotificationRepository)(nil)
