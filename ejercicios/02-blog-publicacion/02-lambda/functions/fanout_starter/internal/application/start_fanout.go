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
	created, err := uc.progress.CreateMeta(ctx, transition, now)
	if err != nil {
		return err
	}
	if !created {
		enqueued, err := uc.progress.InitialJobEnqueued(ctx, transition.EventID)
		if err != nil { return err }
		if enqueued { return nil }
	}

	// crea meta en dynamo apra indicar que inicio el batch de notificacion
	// eventId is the stable identity of this fanout. A retry must reuse it.
	job := domain.FanoutJob{EventID: transition.EventID, FanoutID: transition.EventID, JobID: transition.EventID + "#INITIAL", StreamEventID: transition.EventID, PostID: transition.PostID, AuthorID: transition.AuthorID, BatchNumber: 1, CreatedAt: now}
	//intenta enviar mensaje para que empeice a notificar de a batches
	if err := uc.queue.Send(ctx, job); err != nil {
		return err
	}
	//actualiza estado a encolado (entrego mensaje a sqs)
	return uc.progress.MarkInitialJobEnqueued(ctx, transition.EventID, now)
}
