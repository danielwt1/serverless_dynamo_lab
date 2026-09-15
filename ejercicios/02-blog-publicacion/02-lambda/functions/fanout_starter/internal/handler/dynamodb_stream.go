package handler

import (
	"context"
	"fmt"

	"fanout_starter/internal/domain"
	"github.com/aws/aws-lambda-go/events"
)

type DynamoStreamHandler struct {
	useCase domain.StartFanoutUseCase
}

func NewDynamoStreamHandler(useCase domain.StartFanoutUseCase) *DynamoStreamHandler {
	return &DynamoStreamHandler{useCase: useCase}
}

func (h *DynamoStreamHandler) Handle(ctx context.Context, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		newStatus := record.Change.NewImage["status"].String()
		isPublishedInsert := record.EventName == "INSERT" && newStatus == "PUBLISHED"
		isDraftPublication := record.EventName == "MODIFY" && record.Change.OldImage["status"].String() == "DRAFT" && newStatus == "PUBLISHED"
		if !isPublishedInsert && !isDraftPublication {
			continue
		}
		postIDValue, hasPostID := record.Change.NewImage["post_id"]
		authorIDValue, hasAuthorID := record.Change.NewImage["author_id"]
		postID := postIDValue.String()
		authorID := authorIDValue.String()
		if !hasPostID || !hasAuthorID || postID == "" || authorID == "" {
			return fmt.Errorf("stream record %s is missing post_id or author_id", record.EventID)
		}
		if err := h.useCase.Start(ctx, domain.PublishTransition{EventID: record.EventID, PostID: postID, AuthorID: authorID}); err != nil {
			return fmt.Errorf("start fanout for stream record %s: %w", record.EventID, err)
		}
	}
	return nil
}
