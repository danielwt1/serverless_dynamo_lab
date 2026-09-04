package infraestructure

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"lambda_create_post/internal/domain"
)

type putItemStub struct {
	input *dynamodb.PutItemInput
	err   error
}

func (s *putItemStub) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	s.input = input
	return &dynamodb.PutItemOutput{}, s.err
}

func TestDynamoAdapter_SavePost_StoresDraftInSparseIndex(t *testing.T) {
	client := &putItemStub{}
	post := domain.PostModel{
		PostID: "post-1", AuthorID: "author-1", Status: domain.PostStatusDraft,
		CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z", Description: "draft",
	}

	err := NewDynamoAdapter(client, "posts-table").SavePost(context.Background(), post)
	if err != nil {
		t.Fatalf("SavePost() error = %v", err)
	}
	if client.input == nil || *client.input.TableName != "posts-table" {
		t.Fatalf("PutItem input = %#v", client.input)
	}
	if stringValue(client.input.Item, "GSI1PK") != "STATUS#DRAFT#AUTHOR_ID#author-1" ||
		stringValue(client.input.Item, "GSI1SK") != "CREATED_AT#2026-01-01T00:00:00Z#POST_ID#post-1" {
		t.Errorf("draft index keys = %#v", client.input.Item)
	}
	if _, exists := client.input.Item["published_at"]; exists {
		t.Error("draft must not store published_at")
	}
}

func TestDynamoAdapter_SavePost_StoresPublishedAtWithoutDraftIndex(t *testing.T) {
	client := &putItemStub{}
	post := domain.PostModel{PostID: "post-1", AuthorID: "author-1", Status: domain.PostStatusPublished, CreatedAt: "created", UpdatedAt: "updated", PublishedAt: "published", Description: "post"}

	if err := NewDynamoAdapter(client, "posts-table").SavePost(context.Background(), post); err != nil {
		t.Fatalf("SavePost() error = %v", err)
	}
	if stringValue(client.input.Item, "published_at") != "published" {
		t.Errorf("published_at = %q", stringValue(client.input.Item, "published_at"))
	}
	if _, exists := client.input.Item["GSI1PK"]; exists {
		t.Error("published post must not be stored in the draft sparse index")
	}
}

func TestDynamoAdapter_SavePost_MapsConditionalFailure(t *testing.T) {
	client := &putItemStub{err: &types.ConditionalCheckFailedException{}}
	err := NewDynamoAdapter(client, "posts-table").SavePost(context.Background(), domain.PostModel{})
	if !errors.Is(err, domain.ErrPostAlreadyExists) {
		t.Errorf("error = %v, want ErrPostAlreadyExists", err)
	}
}

func stringValue(item map[string]types.AttributeValue, name string) string {
	value, _ := item[name].(*types.AttributeValueMemberS)
	if value == nil {
		return ""
	}
	return value.Value
}
