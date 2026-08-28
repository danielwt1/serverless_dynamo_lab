package domain

import "context"

type FollowerRepository interface { FindPage(ctx context.Context, authorID, cursor string, limit int32) (FollowerPage, error) }
type ProgressRepository interface {
	LoadCheckpoint(ctx context.Context, eventID string) (FanoutCheckpoint, error)
	RecordPage(ctx context.Context, eventID string, batches []ProgressBatch, nextCursor string, followersCount int, completed bool, updatedAt string) (alreadyRecorded bool, err error)
}
type Queue interface { Send(ctx context.Context, value any) error }
