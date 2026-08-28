package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"follower_fanout_worker/internal/domain"
)

const ( pageSize int32 = 100; recipientsPerJob = 50; maxFollowersPerInvocation = 1000 )

type ProcessFanoutUseCase struct { followers domain.FollowerRepository; progress domain.ProgressRepository; fanoutQueue domain.Queue; notificationQueue domain.Queue; now func() time.Time }
func NewProcessFanoutUseCase(followers domain.FollowerRepository, progress domain.ProgressRepository, fanoutQueue, notificationQueue domain.Queue) *ProcessFanoutUseCase { return &ProcessFanoutUseCase{followers: followers, progress: progress, fanoutQueue: fanoutQueue, notificationQueue: notificationQueue, now: time.Now} }

func (uc *ProcessFanoutUseCase) Process(ctx context.Context, job domain.FanoutJob) error {
	checkpoint, err := uc.progress.LoadCheckpoint(ctx, job.EventID)
	if err != nil { return err }
	if checkpoint.Completed { return nil }
	// META is a durable checkpoint written after every successfully enqueued
	// page. It is preferred over the original SQS cursor on retries.
	cursor, processed := checkpoint.NextCursor, 0
	for processed < maxFollowersPerInvocation {
		page, err := uc.followers.FindPage(ctx, job.AuthorID, cursor, pageSize)
		if err != nil { return err }
		now := uc.now().UTC().Format(time.RFC3339Nano)
		batches := batchesForPage(job.EventID, cursor, page.NextCursor, page.RecipientIDs, now)
		for _, batch := range batches {
			start := batch.part * recipientsPerJob; end := start + recipientsPerJob; if end > len(page.RecipientIDs) { end = len(page.RecipientIDs) }
			if err := uc.notificationQueue.Send(ctx, domain.NotificationJob{EventID: job.EventID, FanoutBatchID: batch.progress.BatchID, PostID: job.PostID, AuthorID: job.AuthorID, RecipientIDs: page.RecipientIDs[start:end], CreatedAt: now}); err != nil { return err }
		}
		processed += len(page.RecipientIDs)
		completed := page.NextCursor == ""
		_, err = uc.progress.RecordPage(ctx, job.EventID, progressBatches(batches), page.NextCursor, len(page.RecipientIDs), completed, now)
		if err != nil { return err }
		if completed { return nil }
		cursor = page.NextCursor
	}
	return uc.fanoutQueue.Send(ctx, domain.FanoutJob{EventID: job.EventID, StreamEventID: job.StreamEventID, PostID: job.PostID, AuthorID: job.AuthorID, Cursor: cursor, BatchNumber: job.BatchNumber + 1, CreatedAt: uc.now().UTC().Format(time.RFC3339Nano)})
}

type pageBatch struct { progress domain.ProgressBatch; part int }
func batchesForPage(eventID, cursorStart, cursorEnd string, recipients []string, createdAt string) []pageBatch { result := make([]pageBatch, 0, (len(recipients)+recipientsPerJob-1)/recipientsPerJob); for start, part := 0, 1; start < len(recipients); start, part = start+recipientsPerJob, part+1 { hash := sha256.Sum256([]byte(cursorStart)); id := eventID+"#CURSOR#"+hex.EncodeToString(hash[:8])+"#PART#"+twoDigits(part); result = append(result, pageBatch{progress: domain.ProgressBatch{BatchID: id, CursorStart: cursorStart, CursorEnd: cursorEnd, RecipientsCount: min(recipientsPerJob, len(recipients)-start), CreatedAt: createdAt}, part: part-1}) }; return result }
func progressBatches(batches []pageBatch) []domain.ProgressBatch { result := make([]domain.ProgressBatch, len(batches)); for i, batch := range batches { result[i] = batch.progress }; return result }
func twoDigits(value int) string { if value < 10 { return "0"+string(rune('0'+value)) }; return string(rune('0'+value/10))+string(rune('0'+value%10)) }
func min(a, b int) int { if a < b { return a }; return b }
