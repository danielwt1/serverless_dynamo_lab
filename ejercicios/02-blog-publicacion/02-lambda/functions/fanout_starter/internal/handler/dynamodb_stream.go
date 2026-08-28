package handler

import (
	"context"
	"fmt"

	"fanout_starter/internal/application"
	"fanout_starter/internal/domain"
	"github.com/aws/aws-lambda-go/events"
)

type DynamoStreamHandler struct {
	useCase *application.StartFanoutUseCase
}

func NewDynamoStreamHandler(useCase *application.StartFanoutUseCase) *DynamoStreamHandler {
	return &DynamoStreamHandler{useCase: useCase}
}

func (h *DynamoStreamHandler) Handle(ctx context.Context, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		if record.EventName != "MODIFY" || record.Change.OldImage["status"].String() != "DRAFT" || record.Change.NewImage["status"].String() != "PUBLISHED" {
			continue
		}
		postID := record.Change.NewImage["post_id"].String()
		authorID := record.Change.NewImage["author_id"].String()
		if postID == "" || authorID == "" {
			return fmt.Errorf("stream record %s is missing post_id or author_id", record.EventID)
		}
		if err := h.useCase.Start(ctx, domain.PublishTransition{EventID: record.EventID, PostID: postID, AuthorID: authorID}); err != nil {
			return fmt.Errorf("start fanout for stream record %s: %w", record.EventID, err)
		}
	}
	return nil
}
