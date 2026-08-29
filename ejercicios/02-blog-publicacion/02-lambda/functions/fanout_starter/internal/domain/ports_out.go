package domain

import "context"

// ProgressRepository persists the progress of a fanout operation.
type ProgressRepository interface {
	CreateMeta(ctx context.Context, transition PublishTransition, createdAt string) (created bool, err error)
	MarkInitialJobEnqueued(ctx context.Context, eventID, updatedAt string) error
}

// FanoutQueue sends the jobs that continue the fanout operation.
type FanoutQueue interface {
	Send(ctx context.Context, job FanoutJob) error
}
