package application

import (
	"context"
	"fanout_starter/internal/domain"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type progressRepositoryStub struct {
	input  *domain.PublishTransition
	err    error
	output bool
}

type fanoutQueueStub struct {
	input *domain.FanoutJob
	err   error
}

func (pr *progressRepositoryStub) CreateMeta(ctx context.Context, transition domain.PublishTransition, createdAt string) (created bool, err error) {
	return pr.output, pr.err

}
func (pr *progressRepositoryStub) MarkInitialJobEnqueued(ctx context.Context, eventID, updatedAt string) error {
	return pr.err
}

func (fn *fanoutQueueStub) Send(ctx context.Context, job domain.FanoutJob) error {
	return fn.err
}

func Test_when_use_case_call_to_creat_and_init_batch_process(t *testing.T) {
	usecasesData := []struct {
		errorExpected       error
		errorExpectedFanout error
		inputPublish        *domain.PublishTransition
		inputQueue          *domain.FanoutJob
		outputPublish       bool
	}{
		{
			errorExpected:       nil,
			errorExpectedFanout: nil,
			inputPublish:        &domain.PublishTransition{},
			inputQueue:          &domain.FanoutJob{},
			outputPublish:       true,
		},
		{
			errorExpected:       nil,
			errorExpectedFanout: fmt.Errorf("error sending message to SQS"),
			inputPublish:        &domain.PublishTransition{},
			inputQueue:          &domain.FanoutJob{},
			outputPublish:       false,
		},
		{
			errorExpected:       fmt.Errorf("error sending message to SQS"),
			errorExpectedFanout: nil,
			inputPublish:        &domain.PublishTransition{},
			inputQueue:          &domain.FanoutJob{},
			outputPublish:       false,
		},
	}
	for _, uc := range usecasesData {
		progressRepo := &progressRepositoryStub{
			input:  uc.inputPublish,
			err:    uc.errorExpected,
			output: uc.outputPublish,
		}
		fanoutQueue := &fanoutQueueStub{
			input: uc.inputQueue,
			err:   uc.errorExpectedFanout,
		}
		usecase := NewStartFanoutUseCase(progressRepo, fanoutQueue)
		err := usecase.Start(context.Background(), *uc.inputPublish)
		if err != nil {
			assert.Error(t, err)
		}

	}

}
