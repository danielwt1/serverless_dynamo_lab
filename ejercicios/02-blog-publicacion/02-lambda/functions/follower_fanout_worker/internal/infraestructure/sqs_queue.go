package infraestructure

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface { SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) }
type SQSQueue struct { client SQSClient; queueURL string }
func NewSQSQueue(client SQSClient, queueURL string) *SQSQueue { return &SQSQueue{client: client, queueURL: queueURL} }
func (q *SQSQueue) Send(ctx context.Context, value any) error { body, err := json.Marshal(value); if err != nil { return err }; _, err = q.client.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl:aws.String(q.queueURL), MessageBody:aws.String(string(body))}); return err }
