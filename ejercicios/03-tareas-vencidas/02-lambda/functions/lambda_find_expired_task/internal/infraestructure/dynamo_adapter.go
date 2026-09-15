package infraestructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"lambda_find_expired_task/domain"
	"lambda_find_expired_task/domain/ports/out"
	"lambda_find_expired_task/internal/infraestructure/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoClientContainer interface {
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type DynamoQueryOverDueAdapter struct {
	tableName string
	client    DynamoClientContainer
}

func NewDynamoQueryOverDueAdapter(client DynamoClientContainer, tablename string) *DynamoQueryOverDueAdapter {
	return &DynamoQueryOverDueAdapter{
		tableName: tablename,
		client:    client,
	}
}
func (dc *DynamoQueryOverDueAdapter) GetOverDueTasks(ctx context.Context, cursor, shardId, executionTime string) (domain.QueryResult, error) {
	initCursor, err := decodeCursor(cursor)
	if err != nil {
		return domain.QueryResult{}, err
	}
	if shardId == "" || executionTime == "" {
		return domain.QueryResult{}, fmt.Errorf("shard_id y execution_time son obligatorios")
	}
	response, err := dc.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &dc.tableName,
		IndexName:              aws.String("task_gsi_pending"),
		KeyConditionExpression: aws.String("#pk = :gsi1pk AND #sk BETWEEN :inicio AND :fin"),
		ExpressionAttributeNames: map[string]string{
			"#pk": "GSI1PK",
			"#sk": "GSI1SK",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{Value: "SHARD#" + shardId + "#STATUS#PENDING"},
			":inicio": &types.AttributeValueMemberS{Value: "EXPIRED_AT#"},
			":fin":    &types.AttributeValueMemberS{Value: "EXPIRED_AT#" + executionTime + "#\uffff"},
		},
		ExclusiveStartKey: initCursor,
		Limit:             aws.Int32(25),
	})
	if err != nil {
		return domain.QueryResult{}, err
	}

	lastEvaluatedKeyString, err := encodeCursor(response.LastEvaluatedKey)
	if err != nil {
		return domain.QueryResult{}, err
	}
	if response.Items == nil || len(response.Items) == 0 {
		return domain.QueryResult{
			Task:   make([]domain.TaskModel, 0),
			Cursor: lastEvaluatedKeyString,
		}, nil
	}
	tasks, err := utils.ConverterDynamo[domain.TaskModel](response.Items)
	if err != nil {
		return domain.QueryResult{}, err
	}

	return domain.QueryResult{
		Task:   tasks,
		Cursor: lastEvaluatedKeyString,
	}, nil

}

func decodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}
	decodedB64Cursor, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	mapValue := make(map[string]cursorAttribute)
	if err := json.Unmarshal(decodedB64Cursor, &mapValue); err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %s", err)
	}
	mapResponse := make(map[string]types.AttributeValue, len(mapValue))
	for key, val := range mapValue {
		switch val.Type {
		case "S":
			mapResponse[key] = &types.AttributeValueMemberS{Value: val.Value}
		case "N":
			mapResponse[key] = &types.AttributeValueMemberN{Value: val.Value}
		case "B":
			binaryValue, err := base64.StdEncoding.DecodeString(val.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to decode binary cursor attribute %q: %w", key, err)
			}
			mapResponse[key] = &types.AttributeValueMemberB{Value: binaryValue}
		default:
			return nil, fmt.Errorf("unsupported cursor attribute type %q", val.Type)
		}
	}
	return mapResponse, nil
}

func encodeCursor(cursor map[string]types.AttributeValue) (string, error) {
	if len(cursor) == 0 {
		return "", nil
	}
	mapValue := make(map[string]cursorAttribute, len(cursor))
	for key, val := range cursor {
		switch typedValue := val.(type) {
		case *types.AttributeValueMemberS:
			mapValue[key] = cursorAttribute{Type: "S", Value: typedValue.Value}
		case *types.AttributeValueMemberN:
			mapValue[key] = cursorAttribute{Type: "N", Value: typedValue.Value}
		case *types.AttributeValueMemberB:
			mapValue[key] = cursorAttribute{Type: "B", Value: base64.StdEncoding.EncodeToString(typedValue.Value)}
		default:
			return "", fmt.Errorf("unsupported cursor attribute %q", key)
		}
	}
	jsonValue, err := json.Marshal(mapValue)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(jsonValue), nil

}

type cursorAttribute struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

var _ out.TaskOverDueRepository = (*DynamoQueryOverDueAdapter)(nil)
