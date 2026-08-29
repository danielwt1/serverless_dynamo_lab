package infraestructure

import (
	"context"
	"fanout_starter/internal/domain"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

type progressDynamoStub struct {
	putItemInput     *dynamodb.PutItemInput
	updateInput      *dynamodb.UpdateItemInput
	putItemOutput    *dynamodb.PutItemOutput
	updateItemOutput *dynamodb.UpdateItemOutput
	err              error
}

func (pd *progressDynamoStub) PutItem(ctx context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	pd.putItemInput = input
	return pd.putItemOutput, pd.err

}
func (pd *progressDynamoStub) UpdateItem(ctx context.Context, input *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	pd.updateInput = input
	return pd.updateItemOutput, pd.err
}

func Test_when_init_process_try_save_dynamo_or_error(t *testing.T) {
	testCases := []struct {
		putItemInput  *dynamodb.PutItemInput
		putItemOutput *dynamodb.PutItemOutput
		errorWanted   error
		expectedErr   error
		transition    domain.PublishTransition
		createdAt     string
		responseOk    bool
	}{
		{
			putItemInput:  &dynamodb.PutItemInput{},
			putItemOutput: &dynamodb.PutItemOutput{},
			errorWanted:   nil,
			transition:    domain.PublishTransition{},
			createdAt:     "2026-01-01T00:00:00Z",
			responseOk:    true,
		},
		{
			putItemInput:  &dynamodb.PutItemInput{},
			putItemOutput: &dynamodb.PutItemOutput{},
			errorWanted:   fmt.Errorf("error saving to DynamoDB"),
			expectedErr:   fmt.Errorf("error saving to DynamoDB"),
			transition:    domain.PublishTransition{},
			createdAt:     "2026-01-01T00:00:00Z",
			responseOk:    false,
		},
		{
			putItemInput:  &dynamodb.PutItemInput{},
			putItemOutput: &dynamodb.PutItemOutput{},
			errorWanted: &types.ConditionalCheckFailedException{
				Message:           aws.String("Conditional check failed"),
				ErrorCodeOverride: aws.String("ConditionalCheckFailedException"),
				Item:              map[string]types.AttributeValue{},
			},
			expectedErr: nil,
			transition:  domain.PublishTransition{},
			createdAt:   "2026-01-01T00:00:00Z",
			responseOk:  false,
		},
	}
	for _, test := range testCases {
		clientMocked := &progressDynamoStub{
			putItemInput:  test.putItemInput,
			putItemOutput: test.putItemOutput,
			err:           test.errorWanted,
		}
		repository := NewProgressRepository(clientMocked, "test-table")
		ok, err := repository.CreateMeta(context.Background(), test.transition, test.createdAt)
		assert.Equal(t, test.responseOk, ok)
		if test.expectedErr == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, test.expectedErr.Error())
		}
	}
}

func Test_when_try_update_status_to_encoled(t *testing.T) {
	testCases := []struct {
		updateInput  *dynamodb.UpdateItemInput
		updateOutput *dynamodb.UpdateItemOutput
		errorWanted  error
		eventId      string
		updatedAt    string
	}{
		{
			updateInput:  &dynamodb.UpdateItemInput{},
			updateOutput: &dynamodb.UpdateItemOutput{},
			errorWanted:  nil,
			eventId:      "test-event-id",
			updatedAt:    "2026-01-01T00:00:00Z",
		},
	}
	for _, test := range testCases {
		clientMocked := &progressDynamoStub{
			updateInput:      test.updateInput,
			updateItemOutput: test.updateOutput,
			err:              test.errorWanted,
		}
		repository := NewProgressRepository(clientMocked, "test-table")
		err := repository.MarkInitialJobEnqueued(context.TODO(), test.eventId, test.updatedAt)
		if test.errorWanted == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, test.errorWanted.Error())
		}
	}
}
