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
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}
type ProgressRepository struct {
	client    TransactionClient
	tableName string
}

func NewProgressRepository(client TransactionClient, tableName string) *ProgressRepository {
	return &ProgressRepository{client: client, tableName: tableName}
}

func (r *ProgressRepository) LoadCheckpoint(ctx context.Context, eventID string) (domain.FanoutCheckpoint, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + eventID},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
	})
	if err != nil {
		return domain.FanoutCheckpoint{}, err
	}
	if len(output.Item) == 0 {
		return domain.FanoutCheckpoint{}, errors.New("fanout progress META not found")
	}
	status, ok := output.Item["status"].(*types.AttributeValueMemberS)
	if !ok {
		return domain.FanoutCheckpoint{}, errors.New("fanout progress META has no status")
	}
	cursor := ""
	if value, ok := output.Item["next_cursor"].(*types.AttributeValueMemberS); ok {
		cursor = value.Value
	}
	return domain.FanoutCheckpoint{NextCursor: cursor, Completed: status.Value == "FANOUT_COMPLETED"}, nil
}

// AcquireLease serializa las instancias de trabajo de un fanout. La escritura condicional es
// la autoridad; la lectura consistente previa solo permite confirmar trabajos completados.
func (r *ProgressRepository) AcquireLease(ctx context.Context, eventID, owner string, nowEpoch, expiresEpoch int64, acquiredAt string) (bool, bool, error) {
	checkpoint, err := r.LoadCheckpoint(ctx, eventID)
	if err != nil {
		return false, false, err
	}
	if checkpoint.Completed {
		return false, true, nil
	}
	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                aws.String(r.tableName),
		Key:                      map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + eventID}, "SK": &types.AttributeValueMemberS{Value: "META"}},
		UpdateExpression:         aws.String("SET lease_owner = :owner, lease_expires_at = :expires, lease_acquired_at = :acquired, updated_at = :acquired ADD lease_version :one"),
		ConditionExpression:      aws.String("#status <> :completed AND (attribute_not_exists(lease_expires_at) OR lease_expires_at < :now)"),
		ExpressionAttributeNames: map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":owner": &types.AttributeValueMemberS{Value: owner}, ":expires": &types.AttributeValueMemberN{Value: itoa64(expiresEpoch)}, ":acquired": &types.AttributeValueMemberS{Value: acquiredAt}, ":completed": &types.AttributeValueMemberS{Value: "FANOUT_COMPLETED"}, ":now": &types.AttributeValueMemberN{Value: itoa64(nowEpoch)}, ":one": &types.AttributeValueMemberN{Value: "1"},
		},
	})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return false, false, nil
	}
	return err == nil, false, err
}

func (r *ProgressRepository) RenewLease(ctx context.Context, eventID, owner string, expiresEpoch int64, updatedAt string) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(r.tableName),
		Key:                       map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + eventID}, "SK": &types.AttributeValueMemberS{Value: "META"}},
		UpdateExpression:          aws.String("SET lease_expires_at = :expires, updated_at = :now"),
		ConditionExpression:       aws.String("lease_owner = :owner AND #status <> :completed"),
		ExpressionAttributeNames:  map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":owner": &types.AttributeValueMemberS{Value: owner}, ":expires": &types.AttributeValueMemberN{Value: itoa64(expiresEpoch)}, ":now": &types.AttributeValueMemberS{Value: updatedAt}, ":completed": &types.AttributeValueMemberS{Value: "FANOUT_COMPLETED"}},
	})
	return err
}

func (r *ProgressRepository) RecordPage(ctx context.Context, eventID, owner string, batches []domain.ProgressBatch, nextCursor string, followersCount int, completed bool, updatedAt string) (bool, error) {
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
	status := "PROCESSING"
	if completed {
		status = "FANOUT_COMPLETED"
	}
	writes = append(writes, types.TransactWriteItem{Update: &types.Update{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		UpdateExpression:         aws.String("SET next_cursor = :cursor, updated_at = :now, #status = :status ADD processed_followers :followers, batches_enqueued :batches, notification_jobs_enqueued :jobs"),
		ConditionExpression:      aws.String("lease_owner = :owner AND #status <> :completed"),
		ExpressionAttributeNames: map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":cursor":    &types.AttributeValueMemberS{Value: nextCursor},
			":now":       &types.AttributeValueMemberS{Value: updatedAt},
			":status":    &types.AttributeValueMemberS{Value: status},
			":owner":     &types.AttributeValueMemberS{Value: owner},
			":completed": &types.AttributeValueMemberS{Value: "FANOUT_COMPLETED"},
			":followers": &types.AttributeValueMemberN{Value: itoa(followersCount)},
			":batches":   &types.AttributeValueMemberN{Value: itoa(len(batches))},
			":jobs":      &types.AttributeValueMemberN{Value: itoa(len(batches))},
		},
	}})
	_, err := r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: writes})
	if err != nil {
		return false, err
	}
	return false, nil
}
func (r *ProgressRepository) ReleaseLease(ctx context.Context, eventID, owner, updatedAt string) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(r.tableName),
		Key:                       map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + eventID}, "SK": &types.AttributeValueMemberS{Value: "META"}},
		UpdateExpression:          aws.String("REMOVE lease_owner, lease_expires_at, lease_acquired_at SET updated_at = :now"),
		ConditionExpression:       aws.String("lease_owner = :owner AND #status <> :completed"),
		ExpressionAttributeNames:  map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":owner": &types.AttributeValueMemberS{Value: owner}, ":now": &types.AttributeValueMemberS{Value: updatedAt}, ":completed": &types.AttributeValueMemberS{Value: "FANOUT_COMPLETED"}},
	})
	return err
}
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	result := ""
	for value > 0 {
		result = string(rune('0'+value%10)) + result
		value /= 10
	}
	return result
}
func itoa64(value int64) string {
	if value == 0 {
		return "0"
	}
	result := ""
	for value > 0 {
		result = string(rune('0'+value%10)) + result
		value /= 10
	}
	return result
}
