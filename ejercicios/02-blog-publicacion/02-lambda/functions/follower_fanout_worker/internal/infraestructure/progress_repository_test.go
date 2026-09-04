package infraestructure

import (
	"context"
	"errors"
	"testing"

	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type progressRepositoryStub struct {
	getInput            *dynamodb.GetItemInput
	outputGetItem       *dynamodb.GetItemOutput
	outputErr           error
	updateInput         *dynamodb.UpdateItemInput
	outputUpdateItem    *dynamodb.UpdateItemOutput
	updateErr           error
	transactInput       *dynamodb.TransactWriteItemsInput
	outputTransactItems *dynamodb.TransactWriteItemsOutput
	transactItemErr     error
}

func (pr *progressRepositoryStub) GetItem(ctx context.Context, input *dynamodb.GetItemInput, options ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	pr.getInput = input
	return pr.outputGetItem, pr.outputErr
}
func (pr *progressRepositoryStub) UpdateItem(ctx context.Context, input *dynamodb.UpdateItemInput, options ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	pr.updateInput = input
	return pr.outputUpdateItem, pr.updateErr
}
func (pr *progressRepositoryStub) TransactWriteItems(ctx context.Context, input *dynamodb.TransactWriteItemsInput, options ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	pr.transactInput = input
	return pr.outputTransactItems, pr.transactItemErr
}

func processingMeta() *dynamodb.GetItemOutput {
	return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
		"status": &types.AttributeValueMemberS{Value: "PROCESSING"},
	}}
}

func TestProgressRepository_AcquireLease(t *testing.T) {
	t.Run("acquires the lease for an active fanout", func(t *testing.T) {
		client := &progressRepositoryStub{outputGetItem: processingMeta(), outputUpdateItem: &dynamodb.UpdateItemOutput{}}
		acquired, completed, err := NewProgressRepository(client, "progress-table").AcquireLease(context.Background(), "event-1", "owner-1", 100, 200, "2026-08-31T00:00:00Z")
		assert.NoError(t, err)
		assert.True(t, acquired)
		assert.False(t, completed)
		assert.NotNil(t, client.updateInput)
	})

	t.Run("does not acquire a completed fanout", func(t *testing.T) {
		client := &progressRepositoryStub{outputGetItem: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"status": &types.AttributeValueMemberS{Value: "FANOUT_COMPLETED"}}}}
		acquired, completed, err := NewProgressRepository(client, "progress-table").AcquireLease(context.Background(), "event-1", "owner-1", 100, 200, "2026-08-31T00:00:00Z")
		assert.NoError(t, err)
		assert.False(t, acquired)
		assert.True(t, completed)
		assert.Nil(t, client.updateInput)
	})

	t.Run("returns no error when another worker holds the lease", func(t *testing.T) {
		client := &progressRepositoryStub{outputGetItem: processingMeta(), updateErr: &types.ConditionalCheckFailedException{}}
		acquired, completed, err := NewProgressRepository(client, "progress-table").AcquireLease(context.Background(), "event-1", "owner-1", 100, 200, "2026-08-31T00:00:00Z")
		assert.NoError(t, err)
		assert.False(t, acquired)
		assert.False(t, completed)
	})
}

func TestProgressRepository_RenewAndReleaseLease(t *testing.T) {
	client := &progressRepositoryStub{outputUpdateItem: &dynamodb.UpdateItemOutput{}}
	repository := NewProgressRepository(client, "progress-table")

	assert.NoError(t, repository.RenewLease(context.Background(), "event-1", "owner-1", 200, "2026-08-31T00:00:00Z"))
	assert.NotNil(t, client.updateInput)
	assert.NoError(t, repository.ReleaseLease(context.Background(), "event-1", "owner-1", "2026-08-31T00:00:00Z"))
	assert.Contains(t, *client.updateInput.UpdateExpression, "REMOVE lease_owner")
}

func TestProgressRepository_RecordPage(t *testing.T) {
	t.Run("writes batch and checkpoint", func(t *testing.T) {
		client := &progressRepositoryStub{outputTransactItems: &dynamodb.TransactWriteItemsOutput{}}
		alreadyRecorded, err := NewProgressRepository(client, "progress-table").RecordPage(context.Background(), "event-1", "owner-1", []domain.ProgressBatch{{BatchID: "batch-1", RecipientsCount: 2}}, "cursor-1", 2, false, "2026-08-31T00:00:00Z")
		assert.NoError(t, err)
		assert.False(t, alreadyRecorded)
		assert.Len(t, client.transactInput.TransactItems, 2)
	})

	t.Run("returns the transaction error", func(t *testing.T) {
		wantErr := errors.New("transaction failed")
		client := &progressRepositoryStub{transactItemErr: wantErr}
		_, err := NewProgressRepository(client, "progress-table").RecordPage(context.Background(), "event-1", "owner-1", nil, "", 0, true, "2026-08-31T00:00:00Z")
		assert.EqualError(t, err, wantErr.Error())
	})
}

func TestProgressRepository_LoadCheckpoint(t *testing.T) {
	testCases := []struct {
		name                string
		outputGetItem       *dynamodb.GetItemOutput
		outputErr           error
		want                domain.FanoutCheckpoint
		wantErr             error
	}{
		{
			name: "returns checkpoint from DynamoDB item",
			outputGetItem: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"status":      &types.AttributeValueMemberS{Value: "PROCESSING"},
				"next_cursor": &types.AttributeValueMemberS{Value: "cursor-1"},
			}},
			outputErr:     nil,
			want:          domain.FanoutCheckpoint{NextCursor: "cursor-1", Completed: false},
		},
		{
			name:          "returns error when DynamoDB fails",
			outputErr:     errors.New("DynamoDB unavailable"),
			wantErr:       errors.New("DynamoDB unavailable"),
		},
		{
			name:          "returns error when META does not exist",
			outputGetItem: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{}},
			wantErr:       errors.New("fanout progress META not found"),
		},
		{
			name: "returns error when META has no status",
			outputGetItem: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"next_cursor": &types.AttributeValueMemberS{Value: "cursor-1"},
			}},
			wantErr: errors.New("fanout progress META has no status"),
		},
		{
			name: "returns completed checkpoint without cursor",
			outputGetItem: &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"status": &types.AttributeValueMemberS{Value: "FANOUT_COMPLETED"},
			}},
			want: domain.FanoutCheckpoint{Completed: true},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			clientStub := &progressRepositoryStub{
				outputGetItem: testCase.outputGetItem,
				outputErr:     testCase.outputErr,
			}
			repository := NewProgressRepository(clientStub, "test-table")

			response, err := repository.LoadCheckpoint(context.Background(), "test-event-id")

			if testCase.wantErr != nil {
				assert.EqualError(t, err, testCase.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.want, response)
			}
		})
	}
}
