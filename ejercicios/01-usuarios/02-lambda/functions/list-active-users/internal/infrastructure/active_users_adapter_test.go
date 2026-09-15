package infrastructure

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"list-active-users/internal/domain"
	"testing"
)

type queryStub struct {
	input  *dynamodb.QueryInput
	output *dynamodb.QueryOutput
	err    error
}

func (s *queryStub) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	s.input = input
	return s.output, s.err
}

func TestActiveUsersAdapter_ListActiveUsers(t *testing.T) {
	client := &queryStub{output: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{{"userId": &types.AttributeValueMemberS{Value: "1"}, "email": &types.AttributeValueMemberS{Value: "ana@example.com"}, "name": &types.AttributeValueMemberS{Value: "Ana"}, "lastName": &types.AttributeValueMemberS{Value: "Perez"}, "state": &types.AttributeValueMemberS{Value: "ACTIVE"}, "created_at": &types.AttributeValueMemberS{Value: "2026-01-01T00:00:00Z"}}}, LastEvaluatedKey: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "USER#1"}, "SK": &types.AttributeValueMemberS{Value: "PROFILE"}}}}
	users, cursor, err := NewActiveUsersAdapter(client, "users-table").ListActiveUsers(context.Background(), 25, "")
	if err != nil {
		t.Fatalf("ListActiveUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Email != "ana@example.com" || cursor == "" {
		t.Errorf("users=%#v cursor=%q", users, cursor)
	}
	if client.input.TableName == nil || *client.input.TableName != "users-table" || client.input.IndexName == nil || *client.input.IndexName != activeUsersIndex || client.input.Limit == nil || *client.input.Limit != 25 {
		t.Errorf("query input = %#v", client.input)
	}
	decoded, err := decodeCursor(cursor)
	if err != nil || attributeString(decoded, "PK") != "USER#1" {
		t.Errorf("cursor decoded=%#v err=%v", decoded, err)
	}
}

func TestActiveUsersAdapter_ListActiveUsers_RejectsInvalidCursor(t *testing.T) {
	_, _, err := NewActiveUsersAdapter(&queryStub{}, "users-table").ListActiveUsers(context.Background(), 1, "not-a-cursor")
	if !errors.Is(err, domain.ErrInvalidCursor) {
		t.Errorf("error = %v", err)
	}
}

func TestActiveUsersAdapter_ListActiveUsers_WrapsClientError(t *testing.T) {
	wantErr := errors.New("timeout")
	_, _, err := NewActiveUsersAdapter(&queryStub{err: wantErr}, "users-table").ListActiveUsers(context.Background(), 1, "")
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	cursor, err := encodeCursor(map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "USER#1"}})
	if err != nil {
		t.Fatal(err)
	}
	key, err := decodeCursor(cursor)
	if err != nil || attributeString(key, "PK") != "USER#1" {
		t.Errorf("key=%#v err=%v", key, err)
	}
}
