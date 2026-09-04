package main

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
		Limit:             aws.Int32(3),
		//ScanIndexForward en false para ordenar de forma descendente
	})
	if err != nil {
		return domain.QueryResult{}, errors.DynamoError
	}
	return getDomain(response)

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
		Limit:             aws.Int32(3),
		ExclusiveStartKey: start,
	})
	if err != nil {
		return domain.QueryResult{}, errors.DynamoError
	}
	return getDomain(result)

}

func getDomain(output *dynamodb.QueryOutput) (domain.QueryResult, error) {
	tasks := make([]domain.TaskModel, 0, len(output.Items))
	cursonCOntinuation, err := util.Encode(output.LastEvaluatedKey)
	if err != nil {
		return domain.QueryResult{}, err
	}
	for _, item := range output.Items {
		var createdAt *time.Time
		var expireAt *time.Time

		if value, ok := item["created_at"].(*types.AttributeValueMemberS); ok {
			parsed, err := time.Parse(time.RFC3339, value.Value)
			if err == nil {
				createdAt = &parsed
			}
		}

		if value, ok := item["updated_at"].(*types.AttributeValueMemberS); ok {
			parsed, err := time.Parse(time.RFC3339, value.Value)
			if err == nil {
				expireAt = &parsed
			}
		}
		task := domain.TaskModel{
			TaskId:      item["task_id"].(*types.AttributeValueMemberS).Value,
			OwnerId:     item["owner_id"].(*types.AttributeValueMemberS).Value,
			CreatedAt:   createdAt,
			Expire_at:   expireAt,
			Description: item["description"].(*types.AttributeValueMemberS).Value,
			Status:      item["status"].(*types.AttributeValueMemberS).Value,
		}
		tasks = append(tasks, task)
	}
	return domain.QueryResult{
		Task:   tasks,
		Cursor: cursonCOntinuation,
	}, nil
}
