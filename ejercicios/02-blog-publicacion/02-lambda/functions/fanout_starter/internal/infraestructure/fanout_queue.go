package infraestructure

import (
	"context"
	"encoding/json"

	"fanout_starter/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}
type FanoutQueue struct {
	client   SQSClient
	queueURL string
}

func NewFanoutQueue(client SQSClient, queueURL string) *FanoutQueue {
	return &FanoutQueue{client: client, queueURL: queueURL}
}
func (q *FanoutQueue) Send(ctx context.Context, job domain.FanoutJob) error {
	body, _ := json.Marshal(job)

	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: aws.String(q.queueURL), MessageBody: aws.String(string(body))})
	if err != nil {
		return err
	}
	return nil

}
