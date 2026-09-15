package infraestructure

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type queryClientStub struct {
	input  *dynamodb.QueryInput
	output *dynamodb.QueryOutput
	err    error
}

func (s *queryClientStub) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	s.input = input
	return s.output, s.err
}

func TestDynamoQueryOverDueAdapterUsesExecutionTimeAndReturnsCursor(t *testing.T) {
	lastKey := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#owner-1"},
		"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-1"},
	}
	client := &queryClientStub{output: &dynamodb.QueryOutput{LastEvaluatedKey: lastKey}}
	adapter := NewDynamoQueryOverDueAdapter(client, "task_data")

	page, err := adapter.GetOverDueTasks(context.Background(), "", "03", "2026-09-14T20:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Task) != 0 || page.Cursor == "" {
		t.Fatalf("página inesperada: %+v", page)
	}
	partition := client.input.ExpressionAttributeValues[":gsi1pk"].(*types.AttributeValueMemberS).Value
	upperBound := client.input.ExpressionAttributeValues[":fin"].(*types.AttributeValueMemberS).Value
	if partition != "SHARD#03#STATUS#PENDING" || upperBound != "EXPIRED_AT#2026-09-14T20:00:00Z#\uffff" {
		t.Fatalf("query incorrecta: partition=%q upperBound=%q", partition, upperBound)
	}

	client.output = &dynamodb.QueryOutput{}
	if _, err := adapter.GetOverDueTasks(context.Background(), page.Cursor, "03", "2026-09-14T20:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(client.input.ExclusiveStartKey, lastKey) {
		t.Fatalf("cursor no reversible: got=%v want=%v", client.input.ExclusiveStartKey, lastKey)
	}
}

func TestCursorRoundTripSupportsDynamoKeyTypes(t *testing.T) {
	want := map[string]types.AttributeValue{
		"string": &types.AttributeValueMemberS{Value: "value"},
		"number": &types.AttributeValueMemberN{Value: "42"},
		"binary": &types.AttributeValueMemberB{Value: []byte{0, 1, 2}},
	}
	encoded, err := encodeCursor(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cursor = %#v, quiere %#v", got, want)
	}
}
