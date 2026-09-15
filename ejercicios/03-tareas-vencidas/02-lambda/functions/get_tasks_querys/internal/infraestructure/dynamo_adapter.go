package infraestructure

import (
	"context"
	"get_task_lambda/internal/domain"
	"get_task_lambda/internal/domain/errors"
	"get_task_lambda/internal/infraestructure/util"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoClientI interface {
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}
type DynamoAdapter struct {
	client    DynamoClientI
	tableName string
}

func NewDynamoAdapter(client DynamoClientI, tableName string) *DynamoAdapter {
	return &DynamoAdapter{
		client:    client,
		tableName: tableName,
	}
}

func (da *DynamoAdapter) GetUserTasks(ctx context.Context, userId string, cursor string) (domain.QueryResult, error) {
	startKey, err := util.Decode(cursor)
	if err != nil {
		return domain.QueryResult{}, err
	}

	response, err := da.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &da.tableName,
		ConsistentRead:         aws.Bool(true),
		KeyConditionExpression: aws.String("#pk = :PK AND begins_with(#sk, :SK)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": "PK",
			"#sk": "SK",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":PK": &types.AttributeValueMemberS{Value: "OWNER_ID#" + userId},
			":SK": &types.AttributeValueMemberS{Value: "TASK_ID#"},
		},
		ExclusiveStartKey: startKey,
		Limit:             aws.Int32(10),
		// ScanIndexForward=false permite ordenar de forma descendente.
	})
	if err != nil {
		return domain.QueryResult{}, errors.DynamoError
	}
	return getDomain(response, fullTaskFromItem)

}
func (da *DynamoAdapter) GetPendingTasksByUser(ctx context.Context, userId string, cursor string) (domain.QueryResult, error) {
	start, err := util.Decode(cursor)
	if err != nil {
		return domain.QueryResult{}, err
	}
	result, err := da.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &da.tableName,
		ConsistentRead:         aws.Bool(true),
		KeyConditionExpression: aws.String("#pk = :PK AND begins_with(#sk, :SK)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": "PK",
			"#sk": "SK",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":PK": &types.AttributeValueMemberS{Value: "OWNER_ID#" + userId},
			":SK": &types.AttributeValueMemberS{Value: "STATUS#PENDING#"},
		},
		Limit:             aws.Int32(10),
		ExclusiveStartKey: start,
	})
	if err != nil {
		return domain.QueryResult{}, errors.DynamoError
	}
	return getDomain(result, pendingTaskFromItem)

}

type taskMapper func(map[string]types.AttributeValue) (domain.TaskModel, error)

func getDomain(output *dynamodb.QueryOutput, mapTask taskMapper) (domain.QueryResult, error) {
	tasks := make([]domain.TaskModel, 0, len(output.Items))
	cursonCOntinuation, err := util.Encode(output.LastEvaluatedKey)
	if err != nil {
		return domain.QueryResult{}, err
	}
	for _, item := range output.Items {
		task, err := mapTask(item)
		if err != nil {
			return domain.QueryResult{}, err
		}
		tasks = append(tasks, task)
	}
	return domain.QueryResult{
		Task:   tasks,
		Cursor: cursonCOntinuation,
	}, nil
}

func fullTaskFromItem(item map[string]types.AttributeValue) (domain.TaskModel, error) {
	status, err := requiredString(item, "status")
	if err != nil {
		return domain.TaskModel{}, err
	}
	return taskFromItem(item, status)
}

func pendingTaskFromItem(item map[string]types.AttributeValue) (domain.TaskModel, error) {
	// Esta proyeccion usa SK=STATUS#PENDING#..., por lo que status no necesita duplicarse.
	return taskFromItem(item, "PENDING")
}

func taskFromItem(item map[string]types.AttributeValue, status string) (domain.TaskModel, error) {
	taskID, err := requiredString(item, "task_id")
	if err != nil {
		return domain.TaskModel{}, err
	}
	ownerID, err := requiredString(item, "owner_id")
	if err != nil {
		return domain.TaskModel{}, err
	}
	description, err := requiredString(item, "description")
	if err != nil {
		return domain.TaskModel{}, err
	}
	return domain.TaskModel{
		TaskId:      taskID,
		OwnerId:     ownerID,
		Description: description,
		CreatedAt:   optionalTime(item, "created_at"),
		Expire_at:   optionalTime(item, "expired_at"),
		Status:      status,
	}, nil
}

func requiredString(item map[string]types.AttributeValue, key string) (string, error) {
	value, ok := item[key].(*types.AttributeValueMemberS)
	if !ok {
		return "", errors.InvalidTask
	}
	return value.Value, nil
}

func optionalTime(item map[string]types.AttributeValue, key string) *time.Time {
	value, ok := item[key].(*types.AttributeValueMemberS)
	if !ok {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value.Value)
	if err != nil {
		return nil
	}
	return &parsed
}
