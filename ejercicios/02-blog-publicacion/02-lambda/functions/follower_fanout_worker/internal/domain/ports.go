package domain

import "context"

type FollowerRepository interface { FindPage(ctx context.Context, authorID, cursor string, limit int32) (FollowerPage, error) }
type ProcessFanoutUseCase interface { Process(ctx context.Context, job FanoutJob, owner string) error }
type ProgressRepository interface {
	LoadCheckpoint(ctx context.Context, eventID string) (FanoutCheckpoint, error)
	AcquireLease(ctx context.Context, eventID, owner string, nowEpoch, expiresEpoch int64, acquiredAt string) (acquired bool, completed bool, err error)
	RenewLease(ctx context.Context, eventID, owner string, expiresEpoch int64, updatedAt string) error
	RecordPage(ctx context.Context, eventID, owner string, batches []ProgressBatch, nextCursor string, followersCount int, completed bool, updatedAt string) (alreadyRecorded bool, err error)
	ReleaseLease(ctx context.Context, eventID, owner, updatedAt string) error
}
type Queue interface { Send(ctx context.Context, value any) error }
