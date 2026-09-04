package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

type SQSHandler struct { useCase domain.ProcessFanoutUseCase }
func NewSQSHandler(useCase domain.ProcessFanoutUseCase) *SQSHandler { return &SQSHandler{useCase: useCase} }
func (h *SQSHandler) Handle(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) { response := events.SQSEventResponse{}; requestID := "unknown"; if lc, ok := lambdacontext.FromContext(ctx); ok { requestID = lc.AwsRequestID }; for _, record := range event.Records { var job domain.FanoutJob; if err := json.Unmarshal([]byte(record.Body), &job); err != nil || job.EventID == "" || job.PostID == "" || job.AuthorID == "" { response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId}); continue }; owner := fmt.Sprintf("%s:%s", requestID, record.MessageId); if err := h.useCase.Process(ctx, job, owner); err != nil { response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId}) } }; return response, nil }
