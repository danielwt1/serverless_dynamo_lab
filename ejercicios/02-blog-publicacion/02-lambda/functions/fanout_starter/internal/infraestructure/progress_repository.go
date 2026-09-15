package infraestructure

import (
	"context"
	"errors"

	"fanout_starter/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ProgressDynamoClient interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type ProgressRepository struct { client ProgressDynamoClient; tableName string }

func NewProgressRepository(client ProgressDynamoClient, tableName string) *ProgressRepository { return &ProgressRepository{client: client, tableName: tableName} }

func (r *ProgressRepository) CreateMeta(ctx context.Context, transition domain.PublishTransition, createdAt string) (bool, error) {
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.tableName), ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"), Item: map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + transition.EventID}, "SK": &types.AttributeValueMemberS{Value: "META"}, "status": &types.AttributeValueMemberS{Value: "PROCESSING"}, "fanout_id": &types.AttributeValueMemberS{Value: transition.EventID}, "post_id": &types.AttributeValueMemberS{Value: transition.PostID}, "author_id": &types.AttributeValueMemberS{Value: transition.AuthorID}, "stream_event_id": &types.AttributeValueMemberS{Value: transition.EventID}, "processed_followers": &types.AttributeValueMemberN{Value: "0"}, "notification_jobs_enqueued": &types.AttributeValueMemberN{Value: "0"}, "batches_enqueued": &types.AttributeValueMemberN{Value: "0"}, "started_at": &types.AttributeValueMemberS{Value: createdAt}, "updated_at": &types.AttributeValueMemberS{Value: createdAt},
	}})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) { return false, nil }
	return err == nil, err
}

// InitialJobEnqueued lets a stream retry distinguish a pending start from one
// already delivered to the fanout queue.
func (r *ProgressRepository) InitialJobEnqueued(ctx context.Context, eventID string) (bool, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + eventID},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
	})
	if err != nil { return false, err }
	value, ok := output.Item["initial_job_enqueued"].(*types.AttributeValueMemberBOOL)
	return ok && value.Value, nil
}

func (r *ProgressRepository) MarkInitialJobEnqueued(ctx context.Context, eventID, updatedAt string) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{TableName: aws.String(r.tableName), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "FANOUT#STREAM_EVENT#" + eventID}, "SK": &types.AttributeValueMemberS{Value: "META"}}, UpdateExpression: aws.String("SET initial_job_enqueued = :true, updated_at = :now"), ExpressionAttributeValues: map[string]types.AttributeValue{":true": &types.AttributeValueMemberBOOL{Value: true}, ":now": &types.AttributeValueMemberS{Value: updatedAt}}})
	return err
}
