package handler

import (
	"context"
	"encoding/json"

	"follower_fanout_worker/internal/application"
	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-lambda-go/events"
)

type SQSHandler struct { useCase *application.ProcessFanoutUseCase }
func NewSQSHandler(useCase *application.ProcessFanoutUseCase) *SQSHandler { return &SQSHandler{useCase: useCase} }
func (h *SQSHandler) Handle(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) { response := events.SQSEventResponse{}; for _, record := range event.Records { var job domain.FanoutJob; if err := json.Unmarshal([]byte(record.Body), &job); err != nil || job.EventID == "" || job.PostID == "" || job.AuthorID == "" { response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId}); continue }; if err := h.useCase.Process(ctx, job); err != nil { response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId}) } }; return response, nil }
