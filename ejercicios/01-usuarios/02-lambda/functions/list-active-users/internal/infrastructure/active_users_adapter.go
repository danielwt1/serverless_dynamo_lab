package infrastructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"list-active-users/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const activeUsersIndex = "GSI1"

type ActiveUsersAdapter struct {
	dynamoClient dynamoQueryAPI
	tableName    string
}

type dynamoQueryAPI interface {
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

func NewActiveUsersAdapter(dynamoClient dynamoQueryAPI, tableName string) *ActiveUsersAdapter {
	return &ActiveUsersAdapter{dynamoClient: dynamoClient, tableName: tableName}
}

func (a *ActiveUsersAdapter) ListActiveUsers(ctx context.Context, limit int32, cursor string) ([]domain.User, string, error) {
	startKey, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}

	log.Printf("dynamodb Query started: table=%s index=%s", a.tableName, activeUsersIndex)
	result, err := a.dynamoClient.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(a.tableName),
		IndexName:                 aws.String(activeUsersIndex),
		KeyConditionExpression:    aws.String("GSI1PK = :activeUsers"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":activeUsers": &types.AttributeValueMemberS{Value: "ACTIVE_USERS"}},
		ExclusiveStartKey:         startKey,
		Limit:                     aws.Int32(limit),
		ScanIndexForward:          aws.Bool(false),
	})
	if err != nil {
		log.Printf("dynamodb Query failed: table=%s index=%s error=%v", a.tableName, activeUsersIndex, err)
		return nil, "", fmt.Errorf("query active users from DynamoDB: %w", err)
	}

	users := make([]domain.User, 0, len(result.Items))
	for _, item := range result.Items {
		users = append(users, domain.User{
			UserID:       attributeString(item, "userId"),
			Email:        attributeString(item, "email"),
			Name:         attributeString(item, "name"),
			LastName:     attributeString(item, "lastName"),
			State:        attributeString(item, "state"),
			RegisteredAt: attributeString(item, "created_at"),
		})
	}

	nextCursor, err := encodeCursor(result.LastEvaluatedKey)
	if err != nil {
		return nil, "", fmt.Errorf("encode DynamoDB cursor: %w", err)
	}
	log.Printf("dynamodb Query completed: table=%s index=%s users=%d has_next_page=%t", a.tableName, activeUsersIndex, len(users), nextCursor != "")

	return users, nextCursor, nil
}

func attributeString(item map[string]types.AttributeValue, name string) string {
	value, ok := item[name].(*types.AttributeValueMemberS)
	if !ok {
		return ""
	}
	return value.Value
}

func encodeCursor(key map[string]types.AttributeValue) (string, error) {
	if len(key) == 0 {
		return "", nil
	}

	values := make(map[string]string, len(key))
	for name, attribute := range key {
		value, ok := attribute.(*types.AttributeValueMemberS)
		if !ok {
			return "", fmt.Errorf("unsupported key attribute %q", name)
		}
		values[name] = value.Value
	}

	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}

	values := map[string]string{}
	if err := json.Unmarshal(payload, &values); err != nil || len(values) == 0 {
		return nil, domain.ErrInvalidCursor
	}

	key := make(map[string]types.AttributeValue, len(values))
	for name, value := range values {
		key[name] = &types.AttributeValueMemberS{Value: value}
	}
	return key, nil
}
