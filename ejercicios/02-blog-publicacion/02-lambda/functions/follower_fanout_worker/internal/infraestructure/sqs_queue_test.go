package infraestructure

import (
	"context"
	"errors"
	"testing"

	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
)

type sqsClientStub struct {
	input *sqs.SendMessageInput
	err   error
}

func (s *sqsClientStub) SendMessage(_ context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	s.input = input
	return &sqs.SendMessageOutput{}, s.err
}

func TestSQSQueue_Send(t *testing.T) {
	t.Run("serializes and sends the job", func(t *testing.T) {
		client := &sqsClientStub{}
		err := NewSQSQueue(client, "https://sqs.example/jobs").Send(context.Background(), domain.FanoutJob{EventID: "event-1"})
		assert.NoError(t, err)
		assert.Equal(t, "https://sqs.example/jobs", *client.input.QueueUrl)
		assert.JSONEq(t, `{"eventId":"event-1","fanoutId":"","jobId":"","streamEventId":"","postId":"","authorId":"","cursor":"","batchNumber":0,"createdAt":""}`, *client.input.MessageBody)
	})

	t.Run("returns the SQS error", func(t *testing.T) {
		wantErr := errors.New("SQS unavailable")
		err := NewSQSQueue(&sqsClientStub{err: wantErr}, "queue-url").Send(context.Background(), domain.FanoutJob{})
		assert.EqualError(t, err, wantErr.Error())
	})

	t.Run("returns a marshal error", func(t *testing.T) {
		err := NewSQSQueue(&sqsClientStub{}, "queue-url").Send(context.Background(), func() {})
		assert.Error(t, err)
	})
}
