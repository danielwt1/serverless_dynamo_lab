package infraestructure

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"lambda_create_post/internal/domain"
	"lambda_create_post/internal/domain/ports/out"
)

type DynamoClient interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type DynamoAdapter struct {
	client    DynamoClient
	tableName string
}

func NewDynamoAdapter(client DynamoClient, tableName string) *DynamoAdapter {
	return &DynamoAdapter{client: client, tableName: tableName}
}

func (a *DynamoAdapter) SavePost(ctx context.Context, post domain.PostModel) error {
	item := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: "AUTHOR_ID#" + post.AuthorID},
		"SK":          &types.AttributeValueMemberS{Value: "STATUS#" + string(post.Status) + "#CREATED_AT#" + post.CreatedAt + "#POST_ID#" + post.PostID},
		"post_id":     &types.AttributeValueMemberS{Value: post.PostID},
		"author_id":   &types.AttributeValueMemberS{Value: post.AuthorID},
		"status":      &types.AttributeValueMemberS{Value: string(post.Status)},
		"created_at":  &types.AttributeValueMemberS{Value: post.CreatedAt},
		"updated_at":  &types.AttributeValueMemberS{Value: post.UpdatedAt},
		"description": &types.AttributeValueMemberS{Value: post.Description},
	}

	if post.Status == domain.PostStatusDraft {
		// draft_gsi is sparse so published posts are not replicated into this index.
		item["GSI1PK"] = &types.AttributeValueMemberS{Value: "STATUS#DRAFT#AUTHOR_ID#" + post.AuthorID}
		item["GSI1SK"] = &types.AttributeValueMemberS{Value: "CREATED_AT#" + post.CreatedAt + "#POST_ID#" + post.PostID}
	} else {
		item["published_at"] = &types.AttributeValueMemberS{Value: post.PublishedAt}
	}

	_, err := a.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(a.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	})
	var conditionalErr *types.ConditionalCheckFailedException
	if errors.As(err, &conditionalErr) {
		return domain.ErrPostAlreadyExists
	}
	return err
}

var _ out.CreatePostPort = (*DynamoAdapter)(nil)
