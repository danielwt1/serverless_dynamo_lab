package infraestructure

import (
	"context"
	"fmt"
	"log"
	"strings"

	"authenticate-user/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// dynamoGetItemAPI is the small part of the DynamoDB client this adapter needs.
// *dynamodb.Client implements it in production; tests can provide a controlled double.
type dynamoGetItemAPI interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

type UserPersistenceAdapter struct {
	dynamoClient dynamoGetItemAPI
	tableName    string
}

func NewUserPersistenceAdapter(dynamoClient dynamoGetItemAPI, tableName string) *UserPersistenceAdapter {
	return &UserPersistenceAdapter{
		dynamoClient,
		tableName,
	}
}

// GetUserByEmail PK & SK son los nombre de las columnas si les hubiera colocado id - metadata ese nombre iria ahi, si tengo SK tmb debe ir en el get
func (up *UserPersistenceAdapter) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	key := "EMAIL#" + email
	log.Printf("dynamodb GetItem started: table=%s", up.tableName)
	item, err := up.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      &up.tableName,
		Key:            map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: key}, "SK": &types.AttributeValueMemberS{Value: "UNIQUE"}},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		log.Printf("dynamodb GetItem failed: table=%s error=%v", up.tableName, err)
		return domain.User{}, fmt.Errorf("get user from DynamoDB: %w", err)
	}
	if item == nil || len(item.Item) == 0 {
		log.Printf("dynamodb GetItem completed: table=%s result=not_found", up.tableName)
		return domain.User{}, domain.UserNotFound
	}
	log.Printf("dynamodb GetItem completed: table=%s result=found", up.tableName)

	pk, err := stringAttribute(item.Item, "PK")
	if err != nil {
		return domain.User{}, err
	}
	if !strings.HasPrefix(pk, "EMAIL#") {
		return domain.User{}, fmt.Errorf("get user from DynamoDB: PK has invalid format")
	}
	password, err := stringAttribute(item.Item, "password")
	if err != nil {
		return domain.User{}, err
	}
	userID, err := stringAttribute(item.Item, "userId")
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{Email: strings.TrimPrefix(pk, "EMAIL#"), Password: password, UserId: userID}, nil
}

func stringAttribute(item map[string]types.AttributeValue, name string) (string, error) {
	attribute, ok := item[name].(*types.AttributeValueMemberS)
	if !ok {
		return "", fmt.Errorf("get user from DynamoDB: attribute %q must be a string", name)
	}
	return attribute.Value, nil
}
