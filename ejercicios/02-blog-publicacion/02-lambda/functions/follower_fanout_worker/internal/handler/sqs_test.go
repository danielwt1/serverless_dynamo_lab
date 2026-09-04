package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

type processFanoutStub struct {
	jobs   []domain.FanoutJob
	owners []string
	err    error
}

func (s *processFanoutStub) Process(_ context.Context, job domain.FanoutJob, owner string) error {
	s.jobs = append(s.jobs, job)
	s.owners = append(s.owners, owner)
	return s.err
}

func TestSQSHandler_Handle(t *testing.T) {
	validBody, err := json.Marshal(domain.FanoutJob{EventID: "event-1", PostID: "post-1", AuthorID: "author-1"})
	if err != nil { t.Fatal(err) }

	t.Run("processes a valid job", func(t *testing.T) {
		useCase := &processFanoutStub{}
		response, err := NewSQSHandler(useCase).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "message-1", Body: string(validBody)}}})
		assert.NoError(t, err)
		assert.Empty(t, response.BatchItemFailures)
		assert.Equal(t, []domain.FanoutJob{{EventID: "event-1", PostID: "post-1", AuthorID: "author-1"}}, useCase.jobs)
		assert.Equal(t, []string{"unknown:message-1"}, useCase.owners)
	})

	t.Run("marks an invalid message as failed", func(t *testing.T) {
		useCase := &processFanoutStub{}
		response, err := NewSQSHandler(useCase).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "message-1", Body: "not-json"}}})
		assert.NoError(t, err)
		assert.Equal(t, []events.SQSBatchItemFailure{{ItemIdentifier: "message-1"}}, response.BatchItemFailures)
		assert.Empty(t, useCase.jobs)
	})

	t.Run("marks a message as failed when the use case fails", func(t *testing.T) {
		useCase := &processFanoutStub{err: errors.New("lease unavailable")}
		response, err := NewSQSHandler(useCase).Handle(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "message-1", Body: string(validBody)}}})
		assert.NoError(t, err)
		assert.Equal(t, []events.SQSBatchItemFailure{{ItemIdentifier: "message-1"}}, response.BatchItemFailures)
	})
}
