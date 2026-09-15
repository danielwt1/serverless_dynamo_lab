package infraestructure

import (
	"context"
	"errors"
	"fmt"
	"lambda_create_task/domain"
	"lambda_create_task/domain/ports/out"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoTransactionClient interface {
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type DynamoTaskRepository struct {
	client    DynamoTransactionClient
	tableName string
}

func NewDynamoTaskRepository(client DynamoTransactionClient, tableName string) *DynamoTaskRepository {
	return &DynamoTaskRepository{client: client, tableName: tableName}
}

func (repository *DynamoTaskRepository) CreatePendingTask(ctx context.Context, task domain.Task) error {
	if repository.client == nil || repository.tableName == "" {
		return errors.New("cliente DynamoDB y tabla son obligatorios")
	}
	if task.TaskID == "" || task.OwnerID == "" || task.ShardID == "" || task.Status != domain.PendingStatus {
		return domain.ErrInvalidTask
	}
	createdAt := task.CreatedAt.UTC().Format(time.RFC3339Nano)
	expiredAt := task.ExpiredAt.UTC().Format(time.RFC3339Nano)
	partitionKey := "OWNER_ID#" + task.OwnerID
	mainItem := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: partitionKey},
		"SK":          &types.AttributeValueMemberS{Value: "TASK_ID#" + task.TaskID},
		"entity_type": &types.AttributeValueMemberS{Value: "TASK"},
		"task_id":     &types.AttributeValueMemberS{Value: task.TaskID},
		"owner_id":    &types.AttributeValueMemberS{Value: task.OwnerID},
		"description": &types.AttributeValueMemberS{Value: task.Description},
		"created_at":  &types.AttributeValueMemberS{Value: createdAt},
		"expired_at":  &types.AttributeValueMemberS{Value: expiredAt},
		"status":      &types.AttributeValueMemberS{Value: domain.PendingStatus},
		"GSI1PK":      &types.AttributeValueMemberS{Value: "SHARD#" + task.ShardID + "#STATUS#PENDING"},
		"GSI1SK":      &types.AttributeValueMemberS{Value: "EXPIRED_AT#" + expiredAt + "#TASK_ID#" + task.TaskID},
	}
	projectionItem := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: partitionKey},
		"SK":          &types.AttributeValueMemberS{Value: pendingProjectionKey(task)},
		"entity_type": &types.AttributeValueMemberS{Value: "TASK_PENDING_PROJECTION"},
		"task_id":     &types.AttributeValueMemberS{Value: task.TaskID},
		"owner_id":    &types.AttributeValueMemberS{Value: task.OwnerID},
		"description": &types.AttributeValueMemberS{Value: task.Description},
		"created_at":  &types.AttributeValueMemberS{Value: createdAt},
		"expired_at":  &types.AttributeValueMemberS{Value: expiredAt},
	}
	_, err := repository.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: conditionalPut(repository.tableName, mainItem)},
			{Put: conditionalPut(repository.tableName, projectionItem)},
		},
	})
	if err != nil {
		var canceled *types.TransactionCanceledException
		if errors.As(err, &canceled) {
			return fmt.Errorf("%w: %v", domain.ErrTaskExists, err)
		}
		return fmt.Errorf("transacción DynamoDB: %w", err)
	}
	return nil
}

func pendingProjectionKey(task domain.Task) string {
	return "STATUS#PENDING#EXPIRED_AT#" + task.ExpiredAt.UTC().Format(time.RFC3339Nano) + "#TASK_ID#" + task.TaskID
}

func conditionalPut(tableName string, item map[string]types.AttributeValue) *types.Put {
	return &types.Put{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	}
}

var _ out.TaskRepository = (*DynamoTaskRepository)(nil)
