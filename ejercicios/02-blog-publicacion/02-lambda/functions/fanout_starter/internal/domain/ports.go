package domain

import "context"

type ProgressRepository interface {
	CreateMeta(ctx context.Context, transition PublishTransition, createdAt string) (created bool, err error)
	MarkInitialJobEnqueued(ctx context.Context, eventID, updatedAt string) error
}

type FanoutQueue interface {
	Send(ctx context.Context, job FanoutJob) error
}
