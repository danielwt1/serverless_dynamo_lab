package infraestructure

import (
	"context"
	"lambda_create_task/domain"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type transactionClientStub struct {
	input *dynamodb.TransactWriteItemsInput
}

func (stub *transactionClientStub) TransactWriteItems(_ context.Context, input *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	stub.input = input
	return &dynamodb.TransactWriteItemsOutput{}, nil
}

func TestCreatePendingTaskWritesMainProjectionAndGSI(t *testing.T) {
	client := &transactionClientStub{}
	repository := NewDynamoTaskRepository(client, "task_data")
	task := domain.Task{TaskID: "task-1", OwnerID: "owner-1", Description: "Pagar", CreatedAt: time.Date(2026, 9, 14, 5, 0, 0, 0, time.UTC), ExpiredAt: time.Date(2026, 9, 20, 5, 0, 0, 0, time.UTC), Status: domain.PendingStatus, ShardID: "03"}

	if err := repository.CreatePendingTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if len(client.input.TransactItems) != 2 {
		t.Fatalf("acciones=%d", len(client.input.TransactItems))
	}
	main := client.input.TransactItems[0].Put.Item
	projection := client.input.TransactItems[1].Put.Item
	if main["GSI1PK"].(*types.AttributeValueMemberS).Value != "SHARD#03#STATUS#PENDING" {
		t.Fatalf("GSI incorrecto: %v", main)
	}
	if projection["SK"].(*types.AttributeValueMemberS).Value != "STATUS#PENDING#EXPIRED_AT#2026-09-20T05:00:00Z#TASK_ID#task-1" {
		t.Fatalf("proyección incorrecta: %v", projection)
	}
}
