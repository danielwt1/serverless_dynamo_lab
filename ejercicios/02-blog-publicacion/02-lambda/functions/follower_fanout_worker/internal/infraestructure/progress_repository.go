package infraestructure

import (
	"context"
	"errors"

	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactionClient interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}
type ProgressRepository struct { client TransactionClient; tableName string }
func NewProgressRepository(client TransactionClient, tableName string) *ProgressRepository { return &ProgressRepository{client: client, tableName: tableName} }

func (r *ProgressRepository) LoadCheckpoint(ctx context.Context, eventID string) (domain.FanoutCheckpoint, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value:"FANOUT#STREAM_EVENT#" + eventID},
			"SK": &types.AttributeValueMemberS{Value:"META"},
		},
	})
	if err != nil { return domain.FanoutCheckpoint{}, err }
	if len(output.Item) == 0 { return domain.FanoutCheckpoint{}, errors.New("fanout progress META not found") }
	status, ok := output.Item["status"].(*types.AttributeValueMemberS)
	if !ok { return domain.FanoutCheckpoint{}, errors.New("fanout progress META has no status") }
	cursor := ""
	if value, ok := output.Item["next_cursor"].(*types.AttributeValueMemberS); ok { cursor = value.Value }
	return domain.FanoutCheckpoint{NextCursor: cursor, Completed: status.Value == "FANOUT_COMPLETED"}, nil
}

func (r *ProgressRepository) RecordPage(ctx context.Context, eventID string, batches []domain.ProgressBatch, nextCursor string, followersCount int, completed bool, updatedAt string) (bool, error) {
	pk := "FANOUT#STREAM_EVENT#" + eventID
	writes := make([]types.TransactWriteItem, 0, len(batches)+1)
	for _, batch := range batches {
		writes = append(writes, types.TransactWriteItem{Put: &types.Put{
			TableName:           aws.String(r.tableName),
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
			Item: map[string]types.AttributeValue{
				"PK":               &types.AttributeValueMemberS{Value: pk},
				"SK":               &types.AttributeValueMemberS{Value: "BATCH#" + batch.BatchID},
				"cursor_start":     &types.AttributeValueMemberS{Value: batch.CursorStart},
				"cursor_end":       &types.AttributeValueMemberS{Value: batch.CursorEnd},
				"recipients_count": &types.AttributeValueMemberN{Value: itoa(batch.RecipientsCount)},
				"status":           &types.AttributeValueMemberS{Value: "ENQUEUED_TO_NOTIFICATION_QUEUE"},
				"created_at":       &types.AttributeValueMemberS{Value: batch.CreatedAt},
			},
		}})
	}
	status := "PROCESSING"; if completed { status = "FANOUT_COMPLETED" }
	writes = append(writes, types.TransactWriteItem{Update: &types.Update{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		UpdateExpression:          aws.String("SET next_cursor = :cursor, updated_at = :now, #status = :status ADD processed_followers :followers, batches_enqueued :batches, notification_jobs_enqueued :jobs"),
		ExpressionAttributeNames:  map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":cursor":    &types.AttributeValueMemberS{Value: nextCursor},
			":now":       &types.AttributeValueMemberS{Value: updatedAt},
			":status":    &types.AttributeValueMemberS{Value: status},
			":followers": &types.AttributeValueMemberN{Value: itoa(followersCount)},
			":batches":   &types.AttributeValueMemberN{Value: itoa(len(batches))},
			":jobs":      &types.AttributeValueMemberN{Value: itoa(len(batches))},
		},
	}})
	_, err := r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems:writes})
	if err == nil { return false, nil }
	var cancelled *types.TransactionCanceledException
	if errors.As(err, &cancelled) { for _, reason := range cancelled.CancellationReasons { if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" { return true, nil } } }
	return false, err
}
func itoa(value int) string { if value == 0 { return "0" }; result := ""; for value > 0 { result = string(rune('0'+value%10))+result; value /= 10 }; return result }
