package application

import (
	"context"
	"time"

	"fanout_starter/internal/domain"
)

type StartFanoutUseCase struct {
	progress domain.ProgressRepository
	queue    domain.FanoutQueue
	now      func() time.Time
}

func NewStartFanoutUseCase(progress domain.ProgressRepository, queue domain.FanoutQueue) *StartFanoutUseCase {
	return &StartFanoutUseCase{progress: progress, queue: queue, now: time.Now}
}

func (uc *StartFanoutUseCase) Start(ctx context.Context, transition domain.PublishTransition) error {
	now := uc.now().UTC().Format(time.RFC3339Nano)
	_, err := uc.progress.CreateMeta(ctx, transition, now)
	if err != nil {
		return err
	}

	// Even when META already exists, send the initial job again. A prior attempt
	// could have failed after persisting META but before SQS accepted the job.
	job := domain.FanoutJob{EventID: transition.EventID, StreamEventID: transition.EventID, PostID: transition.PostID, AuthorID: transition.AuthorID, BatchNumber: 1, CreatedAt: now}
	if err := uc.queue.Send(ctx, job); err != nil {
		return err
	}
	return uc.progress.MarkInitialJobEnqueued(ctx, transition.EventID, now)
}
