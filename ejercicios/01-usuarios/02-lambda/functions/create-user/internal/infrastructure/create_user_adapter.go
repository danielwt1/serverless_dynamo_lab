package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"create-user/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type CreateUserAdapter struct {
	dynamoClient dynamoTransactWriteItemsAPI
	tableName    string
}

type dynamoTransactWriteItemsAPI interface {
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

func NewCreateUserAdapter(dynamoClient dynamoTransactWriteItemsAPI, tableName string) *CreateUserAdapter {
	return &CreateUserAdapter{
		dynamoClient: dynamoClient,
		tableName:    tableName,
	}
}

func (a *CreateUserAdapter) Create(ctx context.Context, user domain.User) error {
	id := uuid.New()
	email := strings.ToLower(strings.TrimSpace(user.Email))
	now := time.Now().UTC().Format(time.RFC3339)

	log.Printf("dynamodb TransactWriteItems started: table=%s", a.tableName)
	_, err := a.dynamoClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(a.tableName),
					Item: map[string]types.AttributeValue{
						"PK":       &types.AttributeValueMemberS{Value: "EMAIL#" + email},
						"SK":       &types.AttributeValueMemberS{Value: "UNIQUE"},
						"password": &types.AttributeValueMemberS{Value: "PASSWORD"},
						"userId":   &types.AttributeValueMemberS{Value: id.String()},
					},
					ConditionExpression: aws.String("attribute_not_exists(PK)"),
				},
			},
			{
				Put: &types.Put{
					TableName: aws.String(a.tableName),
					Item: map[string]types.AttributeValue{
						"PK":          &types.AttributeValueMemberS{Value: "USER#" + id.String()},
						"SK":          &types.AttributeValueMemberS{Value: "PROFILE"},
						"password":    &types.AttributeValueMemberS{Value: "PASSWORD"},
						"name":        &types.AttributeValueMemberS{Value: user.Name},
						"lastName":    &types.AttributeValueMemberS{Value: user.LastName},
						"email":       &types.AttributeValueMemberS{Value: email},
						"dateOfBirth": &types.AttributeValueMemberS{Value: user.DateOfBirth},
						"userId":      &types.AttributeValueMemberS{Value: id.String()},
						"state":       &types.AttributeValueMemberS{Value: "ACTIVE"},
						"GSI1PK":      &types.AttributeValueMemberS{Value: "ACTIVE_USERS"},
						"GSI1SK":      &types.AttributeValueMemberS{Value: now + "#USER#" + id.String()},
						"created_at":  &types.AttributeValueMemberS{Value: now},
						"updated_at":  &types.AttributeValueMemberS{Value: now},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("dynamodb TransactWriteItems failed: table=%s error=%v", a.tableName, err)
		var transactionErr *types.TransactionCanceledException
		if errors.As(err, &transactionErr) {
			for _, reason := range transactionErr.CancellationReasons {
				if aws.ToString(reason.Code) == "ConditionalCheckFailed" {
					log.Printf("dynamodb TransactWriteItems completed: table=%s result=already_exists", a.tableName)
					return domain.ErrUserAlreadyExists
				}
			}
		}

		return fmt.Errorf("create user transaction: %w", err)
	}

	log.Printf("dynamodb TransactWriteItems completed: table=%s result=created", a.tableName)
	return nil
}
