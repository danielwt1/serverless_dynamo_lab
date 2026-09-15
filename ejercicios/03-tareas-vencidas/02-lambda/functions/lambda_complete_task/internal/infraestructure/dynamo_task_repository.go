package infraestructure

import (
	"context"
	"errors"
	"fmt"
	"lambda_complete_task/domain"
	"lambda_complete_task/domain/ports/out"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoTaskClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type DynamoTaskRepository struct {
	client    DynamoTaskClient
	tableName string
}

func NewDynamoTaskRepository(client DynamoTaskClient, tableName string) *DynamoTaskRepository {
	return &DynamoTaskRepository{client: client, tableName: tableName}
}

func (repository *DynamoTaskRepository) CompletePendingTask(ctx context.Context, ownerID, taskID string, completedAt time.Time) (domain.Task, error) {
	if repository.client == nil || repository.tableName == "" {
		return domain.Task{}, errors.New("cliente DynamoDB y tabla son obligatorios")
	}
	partitionKey := "OWNER_ID#" + ownerID
	sortKey := "TASK_ID#" + taskID
	result, err := repository.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(repository.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: partitionKey},
			"SK": &types.AttributeValueMemberS{Value: sortKey},
		},
	})
	if err != nil {
		return domain.Task{}, fmt.Errorf("leer tarea: %w", err)
	}
	if len(result.Item) == 0 {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	task, err := taskFromItem(result.Item)
	if err != nil {
		return domain.Task{}, err
	}
	if task.Status == domain.CompletedStatus {
		return task, nil
	}
	if task.Status != domain.PendingStatus {
		return domain.Task{}, domain.ErrTaskNotPending
	}

	completedAt = completedAt.UTC()
	projectionKey := "STATUS#PENDING#EXPIRED_AT#" + task.ExpiredAt.UTC().Format(time.RFC3339Nano) + "#TASK_ID#" + task.TaskID
	_, err = repository.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Update: &types.Update{
				TableName:           aws.String(repository.tableName),
				Key:                 map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: partitionKey}, "SK": &types.AttributeValueMemberS{Value: sortKey}},
				ConditionExpression: aws.String("#status = :pending"),
				UpdateExpression:    aws.String("SET #status = :completed, completed_at = :completed_at REMOVE GSI1PK, GSI1SK"),
				ExpressionAttributeNames: map[string]string{
					"#status": "status",
				},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":pending":      &types.AttributeValueMemberS{Value: domain.PendingStatus},
					":completed":    &types.AttributeValueMemberS{Value: domain.CompletedStatus},
					":completed_at": &types.AttributeValueMemberS{Value: completedAt.Format(time.RFC3339Nano)},
				},
			}},
			{Delete: &types.Delete{
				TableName: aws.String(repository.tableName),
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: partitionKey},
					"SK": &types.AttributeValueMemberS{Value: projectionKey},
				},
			}},
		},
	})
	if err != nil {
		var canceled *types.TransactionCanceledException
		if errors.As(err, &canceled) {
			return domain.Task{}, fmt.Errorf("%w: %v", domain.ErrTaskNotPending, err)
		}
		return domain.Task{}, fmt.Errorf("transacción DynamoDB: %w", err)
	}
	task.Status = domain.CompletedStatus
	task.CompletedAt = &completedAt
	return task, nil
}

func taskFromItem(item map[string]types.AttributeValue) (domain.Task, error) {
	stringValue := func(key string) (string, error) {
		value, ok := item[key].(*types.AttributeValueMemberS)
		if !ok || value.Value == "" {
			return "", fmt.Errorf("atributo %s ausente o inválido", key)
		}
		return value.Value, nil
	}
	taskID, err := stringValue("task_id")
	if err != nil {
		return domain.Task{}, err
	}
	ownerID, err := stringValue("owner_id")
	if err != nil {
		return domain.Task{}, err
	}
	description, err := stringValue("description")
	if err != nil {
		return domain.Task{}, err
	}
	status, err := stringValue("status")
	if err != nil {
		return domain.Task{}, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, attributeString(item, "created_at"))
	if err != nil {
		return domain.Task{}, fmt.Errorf("created_at inválido: %w", err)
	}
	expiredAt, err := time.Parse(time.RFC3339Nano, attributeString(item, "expired_at"))
	if err != nil {
		return domain.Task{}, fmt.Errorf("expired_at inválido: %w", err)
	}
	task := domain.Task{TaskID: taskID, OwnerID: ownerID, Description: description, CreatedAt: createdAt, ExpiredAt: expiredAt, Status: status}
	if value := attributeString(item, "completed_at"); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339Nano, value)
		if parseErr != nil {
			return domain.Task{}, fmt.Errorf("completed_at inválido: %w", parseErr)
		}
		task.CompletedAt = &parsed
	}
	return task, nil
}

func attributeString(item map[string]types.AttributeValue, key string) string {
	value, _ := item[key].(*types.AttributeValueMemberS)
	if value == nil {
		return ""
	}
	return value.Value
}

var _ out.TaskRepository = (*DynamoTaskRepository)(nil)
