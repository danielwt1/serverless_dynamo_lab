package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"fanout_starter/internal/domain"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type startFanoutUseCaseStub struct {
	transitions []domain.PublishTransition
	err         error
}

func (s *startFanoutUseCaseStub) Start(_ context.Context, transition domain.PublishTransition) error {
	s.transitions = append(s.transitions, transition)
	return s.err
}

func dynamoDBEventFromJSON(t *testing.T, body string) events.DynamoDBEvent {
	t.Helper()

	var event events.DynamoDBEvent
	require.NoError(t, json.Unmarshal([]byte(body), &event))
	return event
}

func TestDynamoStreamHandler_Handle_StartsFanoutForPublishedPost(t *testing.T) {
	useCase := &startFanoutUseCaseStub{}
	handler := NewDynamoStreamHandler(useCase)
	event := dynamoDBEventFromJSON(t, `{
		"Records": [{
			"eventID": "stream-event-1",
			"eventName": "MODIFY",
			"dynamodb": {
				"OldImage": {"status": {"S": "DRAFT"}},
				"NewImage": {
					"status": {"S": "PUBLISHED"},
					"post_id": {"S": "post-1"},
					"author_id": {"S": "author-1"}
				}
			}
		}]
	}`)

	err := handler.Handle(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, []domain.PublishTransition{{
		EventID:  "stream-event-1",
		PostID:   "post-1",
		AuthorID: "author-1",
	}}, useCase.transitions)
}

func TestDynamoStreamHandler_Handle_IgnoresUnrelatedChanges(t *testing.T) {
	useCase := &startFanoutUseCaseStub{}
	handler := NewDynamoStreamHandler(useCase)
	event := dynamoDBEventFromJSON(t, `{
		"Records": [{
			"eventID": "stream-event-1",
			"eventName": "MODIFY",
			"dynamodb": {
				"OldImage": {"status": {"S": "DRAFT"}},
				"NewImage": {"status": {"S": "DRAFT"}}
			}
		}]
	}`)

	err := handler.Handle(context.Background(), event)

	require.NoError(t, err)
	assert.Empty(t, useCase.transitions)
}

func TestDynamoStreamHandler_Handle_ReturnsErrorWhenPostDataIsMissing(t *testing.T) {
	useCase := &startFanoutUseCaseStub{}
	handler := NewDynamoStreamHandler(useCase)
	event := dynamoDBEventFromJSON(t, `{
		"Records": [{
			"eventID": "stream-event-1",
			"eventName": "MODIFY",
			"dynamodb": {
				"OldImage": {"status": {"S": "DRAFT"}},
				"NewImage": {
					"status": {"S": "PUBLISHED"},
					"author_id": {"S": "author-1"}
				}
			}
		}]
	}`)

	err := handler.Handle(context.Background(), event)

	assert.EqualError(t, err, "stream record stream-event-1 is missing post_id or author_id")
	assert.Empty(t, useCase.transitions)
}

func TestDynamoStreamHandler_Handle_WrapsUseCaseError(t *testing.T) {
	wantErr := errors.New("queue unavailable")
	useCase := &startFanoutUseCaseStub{err: wantErr}
	handler := NewDynamoStreamHandler(useCase)
	event := dynamoDBEventFromJSON(t, `{
		"Records": [{
			"eventID": "stream-event-1",
			"eventName": "MODIFY",
			"dynamodb": {
				"OldImage": {"status": {"S": "DRAFT"}},
				"NewImage": {
					"status": {"S": "PUBLISHED"},
					"post_id": {"S": "post-1"},
					"author_id": {"S": "author-1"}
				}
			}
		}]
	}`)

	err := handler.Handle(context.Background(), event)

	assert.ErrorIs(t, err, wantErr)
	assert.Len(t, useCase.transitions, 1)
}
