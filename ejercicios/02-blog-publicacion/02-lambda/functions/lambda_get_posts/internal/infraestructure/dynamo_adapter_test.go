package infraestructure

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"lambda_posts_mutable/internal/domain"
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

func TestDynamoAdapter_GetPublishedPosts_BuildsQueryAndMapsPage(t *testing.T) {
	client := &queryStub{output: &dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{postItem("author-1", "post-1", "published")},
		LastEvaluatedKey: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "AUTHOR_ID#author-1"},
			"SK": &types.AttributeValueMemberS{Value: "STATUS#PUBLISHED#CREATED_AT#created#POST_ID#post-1"},
		},
	}}

	page, err := NewDynamoAdapter(client, "posts-table").GetPublishedPosts(context.Background(), "author-1", "PUBLISHED", "", 25)
	if err != nil {
		t.Fatalf("GetPublishedPosts() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].AuthorId != "author-1" || page.Items[0].PublishedAt != "published" || page.NextCursor == "" {
		t.Errorf("page = %+v", page)
	}
	if client.input == nil || *client.input.TableName != "posts-table" || *client.input.Limit != 25 || attributeValue(client.input.ExpressionAttributeValues, ":pk_val") != "AUTHOR_ID#author-1" || attributeValue(client.input.ExpressionAttributeValues, ":status") != "STATUS#PUBLISHED" {
		t.Errorf("query input = %#v", client.input)
	}
}

func TestDynamoAdapter_GetDraftPosts_UsesSparseIndex(t *testing.T) {
	client := &queryStub{output: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{postItem("author-1", "post-1", "")}}}

	page, err := NewDynamoAdapter(client, "posts-table").GetDraftPosts(context.Background(), "author-1", "", 10)
	if err != nil {
		t.Fatalf("GetDraftPosts() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].PublishedAt != "" {
		t.Errorf("page = %+v", page)
	}
	if client.input == nil || client.input.IndexName == nil || *client.input.IndexName != "draft_gsi" || attributeValue(client.input.ExpressionAttributeValues, ":draftAuthor") != "STATUS#DRAFT#AUTHOR_ID#author-1" {
		t.Errorf("query input = %#v", client.input)
	}
}

func TestDynamoAdapter_GetPublishedPosts_RejectsInvalidCursorBeforeQuery(t *testing.T) {
	client := &queryStub{}
	_, err := NewDynamoAdapter(client, "posts-table").GetPublishedPosts(context.Background(), "author-1", "PUBLISHED", "invalid", 10)
	if !errors.Is(err, domain.ErrInvalidCursor) || client.input != nil {
		t.Errorf("error=%v input=%#v", err, client.input)
	}
}

func TestDynamoAdapter_GetDraftPosts_ReturnsDynamoError(t *testing.T) {
	wantErr := errors.New("DynamoDB unavailable")
	client := &queryStub{err: wantErr}
	_, err := NewDynamoAdapter(client, "posts-table").GetDraftPosts(context.Background(), "author-1", "", 10)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	cursor, err := encodeCursor(map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "AUTHOR_ID#author-1"}})
	if err != nil {
		t.Fatal(err)
	}
	key, err := decodeCursor(cursor)
	if err != nil || attributeValue(key, "PK") != "AUTHOR_ID#author-1" {
		t.Errorf("key=%#v err=%v", key, err)
	}
}

func postItem(authorID, postID, publishedAt string) map[string]types.AttributeValue {
	item := map[string]types.AttributeValue{
		"PK":          &types.AttributeValueMemberS{Value: "AUTHOR_ID#" + authorID},
		"post_id":     &types.AttributeValueMemberS{Value: postID},
		"created_at":  &types.AttributeValueMemberS{Value: "created"},
		"updated_at":  &types.AttributeValueMemberS{Value: "updated"},
		"description": &types.AttributeValueMemberS{Value: "description"},
	}
	if publishedAt != "" {
		item["published_at"] = &types.AttributeValueMemberS{Value: publishedAt}
	}
	return item
}

func attributeValue(item map[string]types.AttributeValue, name string) string {
	value, _ := item[name].(*types.AttributeValueMemberS)
	if value == nil {
		return ""
	}
	return value.Value
}
