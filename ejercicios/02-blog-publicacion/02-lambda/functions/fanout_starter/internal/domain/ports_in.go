package domain

import "context"

// StartFanoutUseCase is the input port used by the DynamoDB stream handler.
type StartFanoutUseCase interface {
	Start(ctx context.Context, transition PublishTransition) error
}
