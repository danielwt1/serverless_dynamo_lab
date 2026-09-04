package infraestructure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authenticate-user/internal/domain"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoGetItemStub struct {
	output *dynamodb.GetItemOutput
	err    error
	input  *dynamodb.GetItemInput
}

func (s *dynamoGetItemStub) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	s.input = input
	return s.output, s.err
}

func TestUserPersistenceAdapter_GetUserByEmail(t *testing.T) {
	tests := []struct {
		name      string
		output    *dynamodb.GetItemOutput
		clientErr error
		wantUser  domain.User
		wantErr   error
	}{
		{
			name: "maps a DynamoDB item to a domain user",
			output: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"PK":       &types.AttributeValueMemberS{Value: "EMAIL#ana@example.com"},
				"password": &types.AttributeValueMemberS{Value: "secret"},
				"userId":   &types.AttributeValueMemberS{Value: "user-123"},
			}},
			wantUser: domain.User{Email: "ana@example.com", Password: "secret", UserId: "user-123"},
		},
		{
			name: "returns an error when the primary key attribute is missing",
			output: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"password": &types.AttributeValueMemberS{Value: "secret"},
				"userId":   &types.AttributeValueMemberS{Value: "user-123"},
			}},
			wantErr: errors.New("attribute \"PK\" must be a string"),
		},
		{name: "returns not found for an empty item", output: &dynamodb.GetItemOutput{}, wantErr: domain.UserNotFound},
		{name: "returns not found for an empty DynamoDB response", wantErr: domain.UserNotFound},
		{name: "wraps a client error", clientErr: errors.New("DynamoDB timeout"), wantErr: errors.New("DynamoDB timeout")},
		{
			name: "returns an error when the primary key has an invalid format",
			output: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"PK":       &types.AttributeValueMemberS{Value: "ana@example.com"},
				"password": &types.AttributeValueMemberS{Value: "secret"},
				"userId":   &types.AttributeValueMemberS{Value: "user-123"},
			}},
			wantErr: errors.New("PK has invalid format"),
		},
		{
			name: "returns an error for a malformed item instead of panicking",
			output: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "EMAIL#ana@example.com"},
			}},
			wantErr: errors.New("attribute \"password\" must be a string"),
		},
		{
			name: "returns an error when the user identifier attribute is missing",
			output: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"PK":       &types.AttributeValueMemberS{Value: "EMAIL#ana@example.com"},
				"password": &types.AttributeValueMemberS{Value: "secret"},
			}},
			wantErr: errors.New("attribute \"userId\" must be a string"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &dynamoGetItemStub{output: tt.output, err: tt.clientErr}
			adapter := NewUserPersistenceAdapter(client, "users-table")

			user, err := adapter.GetUserByEmail(context.Background(), "ana@example.com")

			if tt.wantErr != nil {
				if err == nil || !containsError(err, tt.wantErr) {
					t.Fatalf("GetUserByEmail() error = %v, want an error containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("GetUserByEmail() error = %v, want nil", err)
			}
			if user != tt.wantUser {
				t.Errorf("GetUserByEmail() user = %#v, want %#v", user, tt.wantUser)
			}
			assertGetItemInput(t, client.input)
		})
	}
}

func containsError(err, want error) bool {
	return errors.Is(err, want) || (err != nil && want != nil && strings.Contains(err.Error(), want.Error()))
}

func assertGetItemInput(t *testing.T, input *dynamodb.GetItemInput) {
	t.Helper()
	if input == nil {
		t.Fatal("GetItem() was not called")
	}
	if input.TableName == nil || *input.TableName != "users-table" {
		t.Errorf("GetItem() table = %v, want users-table", input.TableName)
	}
	if input.ConsistentRead == nil || !*input.ConsistentRead {
		t.Error("GetItem() ConsistentRead = false, want true")
	}
	pk, pkOK := input.Key["PK"].(*types.AttributeValueMemberS)
	sk, skOK := input.Key["SK"].(*types.AttributeValueMemberS)
	if !pkOK || pk.Value != "EMAIL#ana@example.com" {
		t.Errorf("GetItem() PK = %#v, want EMAIL#ana@example.com", input.Key["PK"])
	}
	if !skOK || sk.Value != "UNIQUE" {
		t.Errorf("GetItem() SK = %#v, want UNIQUE", input.Key["SK"])
	}
}
