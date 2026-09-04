package infraestructure

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
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

func TestFollowerRepository_FindPage(t *testing.T) {
	t.Run("returns followers and next cursor", func(t *testing.T) {
		client := &queryClientStub{output: &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{"SK": &types.AttributeValueMemberS{Value: "FOLLOWER#user-1"}},
				{"SK": &types.AttributeValueMemberS{Value: "FOLLOWER#user-2"}},
			},
			LastEvaluatedKey: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "FOLLOWING#author-1"}},
		}}

		page, err := NewFollowerRepository(client, "followers-table").FindPage(context.Background(), "author-1", "", 25)

		assert.NoError(t, err)
		assert.Equal(t, []string{"user-1", "user-2"}, page.RecipientIDs)
		assert.NotEmpty(t, page.NextCursor)
		assert.Equal(t, "followers-table", *client.input.TableName)
		assert.Equal(t, int32(25), *client.input.Limit)
	})

	t.Run("returns an error for an invalid cursor", func(t *testing.T) {
		client := &queryClientStub{}
		_, err := NewFollowerRepository(client, "followers-table").FindPage(context.Background(), "author-1", "invalid", 25)
		assert.EqualError(t, err, "invalid cursor")
		assert.Nil(t, client.input)
	})

	t.Run("returns the client error", func(t *testing.T) {
		wantErr := errors.New("DynamoDB unavailable")
		_, err := NewFollowerRepository(&queryClientStub{err: wantErr}, "followers-table").FindPage(context.Background(), "author-1", "", 25)
		assert.EqualError(t, err, wantErr.Error())
	})

	t.Run("rejects an invalid follower item", func(t *testing.T) {
		client := &queryClientStub{output: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{{"SK": &types.AttributeValueMemberS{Value: "POST#1"}}}}}
		_, err := NewFollowerRepository(client, "followers-table").FindPage(context.Background(), "author-1", "", 25)
		assert.EqualError(t, err, "invalid follower item")
	})
}
