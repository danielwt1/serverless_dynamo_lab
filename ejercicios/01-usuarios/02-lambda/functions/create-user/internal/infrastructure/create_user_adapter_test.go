package infrastructure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"create-user/internal/domain"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type transactWriteStub struct { input *dynamodb.TransactWriteItemsInput; err error }
func (s *transactWriteStub) TransactWriteItems(_ context.Context, input *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) { s.input = input; return &dynamodb.TransactWriteItemsOutput{}, s.err }

func TestCreateUserAdapter_Create(t *testing.T) {
	client := &transactWriteStub{}
	err := NewCreateUserAdapter(client, "users-table").Create(context.Background(), domain.User{Name: "Ana", LastName: "Perez", Email: " ANA@Example.COM ", DateOfBirth: "2000-01-02"})
	if err != nil { t.Fatalf("Create() error = %v", err) }
	if client.input == nil || len(client.input.TransactItems) != 2 { t.Fatalf("transaction items = %#v, want two items", client.input) }
	unique := client.input.TransactItems[0].Put
	profile := client.input.TransactItems[1].Put
	if value(unique.Item, "PK") != "EMAIL#ana@example.com" || value(unique.Item, "SK") != "UNIQUE" { t.Errorf("unique item = %#v", unique.Item) }
	if unique.TableName == nil || *unique.TableName != "users-table" || unique.ConditionExpression == nil || *unique.ConditionExpression != "attribute_not_exists(PK)" { t.Errorf("unique put configuration = %#v", unique) }
	id := value(unique.Item, "userId")
	if value(profile.Item, "PK") != "USER#"+id || value(profile.Item, "email") != "ana@example.com" || value(profile.Item, "state") != "ACTIVE" || value(profile.Item, "GSI1PK") != "ACTIVE_USERS" { t.Errorf("profile item = %#v", profile.Item) }
}

func TestCreateUserAdapter_Create_WrapsClientError(t *testing.T) {
	wantErr := errors.New("timeout")
	err := NewCreateUserAdapter(&transactWriteStub{err: wantErr}, "users-table").Create(context.Background(), domain.User{})
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "create user transaction") { t.Errorf("Create() error = %v", err) }
}

func value(item map[string]types.AttributeValue, key string) string { value, _ := item[key].(*types.AttributeValueMemberS); if value == nil { return "" }; return value.Value }
