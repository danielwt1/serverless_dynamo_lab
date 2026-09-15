package infraestructure

import (
	"context"
	"lambda_complete_task/domain"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type taskClientStub struct {
	item  map[string]types.AttributeValue
	write *dynamodb.TransactWriteItemsInput
}

func (stub *taskClientStub) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return &dynamodb.GetItemOutput{Item: stub.item}, nil
}

func (stub *taskClientStub) TransactWriteItems(_ context.Context, input *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	stub.write = input
	return &dynamodb.TransactWriteItemsOutput{}, nil
}

func TestCompletePendingTaskUpdatesMainAndDeletesProjection(t *testing.T) {
	client := &taskClientStub{item: pendingItem()}
	repository := NewDynamoTaskRepository(client, "task_data")
	task, err := repository.CompletePendingTask(context.Background(), "owner-1", "task-1", time.Date(2026, 9, 15, 5, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.CompletedStatus || task.CompletedAt == nil || len(client.write.TransactItems) != 2 {
		t.Fatalf("resultado incorrecto: task=%+v write=%+v", task, client.write)
	}
	deletedKey := client.write.TransactItems[1].Delete.Key["SK"].(*types.AttributeValueMemberS).Value
	if deletedKey != "STATUS#PENDING#EXPIRED_AT#2026-09-20T05:00:00Z#TASK_ID#task-1" {
		t.Fatalf("proyección eliminada=%q", deletedKey)
	}
}

func TestCompleteTaskIsIdempotentWhenAlreadyCompleted(t *testing.T) {
	item := pendingItem()
	item["status"] = &types.AttributeValueMemberS{Value: domain.CompletedStatus}
	client := &taskClientStub{item: item}
	if _, err := NewDynamoTaskRepository(client, "task_data").CompletePendingTask(context.Background(), "owner-1", "task-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if client.write != nil {
		t.Fatal("no debe repetir la transacción")
	}
}

func pendingItem() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"task_id": &types.AttributeValueMemberS{Value: "task-1"}, "owner_id": &types.AttributeValueMemberS{Value: "owner-1"},
		"description": &types.AttributeValueMemberS{Value: "Pagar"}, "created_at": &types.AttributeValueMemberS{Value: "2026-09-14T05:00:00Z"},
		"expired_at": &types.AttributeValueMemberS{Value: "2026-09-20T05:00:00Z"}, "status": &types.AttributeValueMemberS{Value: domain.PendingStatus},
	}
}
