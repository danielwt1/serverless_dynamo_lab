package infraestructure

import (
	"context"
	"fanout_starter/internal/domain"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
)

type sqsClientStub struct {
	input  *sqs.SendMessageInput
	err    error
	output *sqs.SendMessageOutput
}

func (sqs *sqsClientStub) SendMessage(ctx context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	sqs.input = input
	return sqs.output, sqs.err
}

func Test_when_system_try_send_message(t *testing.T) {
	//declare mock
	testsData := []struct {
		output     *sqs.SendMessageOutput
		wantErr    error
		queueUrl   string
		input      *sqs.SendMessageInput
		inputModel domain.FanoutJob
	}{
		{
			output:     &sqs.SendMessageOutput{},
			wantErr:    nil,
			queueUrl:   "https://sqs.usuario.com",
			input:      &sqs.SendMessageInput{},
			inputModel: domain.FanoutJob{},
		},
		{
			output:     &sqs.SendMessageOutput{},
			wantErr:    fmt.Errorf("error sending message to SQS"),
			queueUrl:   "https://sqs.usuario.com",
			input:      &sqs.SendMessageInput{},
			inputModel: domain.FanoutJob{},
		},
	}
	for _, test := range testsData {
		clientMocked := &sqsClientStub{
			output: test.output,
			err:    test.wantErr,
			input:  test.input,
		}
		adapter := NewFanoutQueue(clientMocked, test.queueUrl)
		err := adapter.Send(context.Background(), test.inputModel)
		if err != nil {
			assert.Equal(t, test.wantErr, err)

		} else {
			assert.NoError(t, err)
		}

	}

}
