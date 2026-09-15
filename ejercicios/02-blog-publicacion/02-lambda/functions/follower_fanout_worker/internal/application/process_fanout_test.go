package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"follower_fanout_worker/internal/domain"
	"github.com/stretchr/testify/assert"
)

type followerRepositoryStub struct {
	page  domain.FollowerPage
	err   error
	calls int
}

func (s *followerRepositoryStub) FindPage(_ context.Context, _ string, _ string, _ int32) (domain.FollowerPage, error) {
	s.calls++
	return s.page, s.err
}

type progressRepositoryStub struct {
	acquired  bool
	completed bool
	err       error
	checkpoint domain.FanoutCheckpoint
	records   int
	releases  int
	renewals  int
}

func (s *progressRepositoryStub) LoadCheckpoint(context.Context, string) (domain.FanoutCheckpoint, error) { return s.checkpoint, s.err }
func (s *progressRepositoryStub) AcquireLease(context.Context, string, string, int64, int64, string) (bool, bool, error) { return s.acquired, s.completed, s.err }
func (s *progressRepositoryStub) RenewLease(context.Context, string, string, int64, string) error { s.renewals++; return s.err }
func (s *progressRepositoryStub) RecordPage(context.Context, string, string, []domain.ProgressBatch, string, int, bool, string) (bool, error) { s.records++; return false, s.err }
func (s *progressRepositoryStub) ReleaseLease(context.Context, string, string, string) error { s.releases++; return s.err }

type queueStub struct {
	values []any
	err    error
}

func (s *queueStub) Send(_ context.Context, value any) error { s.values = append(s.values, value); return s.err }

func TestProcessFanoutUseCase_Process(t *testing.T) {
	fixedNow := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	job := domain.FanoutJob{EventID: "event-1", PostID: "post-1", AuthorID: "author-1"}

	t.Run("sends notifications and records a completed page", func(t *testing.T) {
		followers := &followerRepositoryStub{page: domain.FollowerPage{RecipientIDs: []string{"user-1", "user-2"}}}
		progress := &progressRepositoryStub{acquired: true}
		fanoutQueue, notificationQueue := &queueStub{}, &queueStub{}
		useCase := NewProcessFanoutUseCase(followers, progress, fanoutQueue, notificationQueue, time.Minute)
		useCase.now = func() time.Time { return fixedNow }

		err := useCase.Process(context.Background(), job, "owner-1")

		assert.NoError(t, err)
		assert.Equal(t, 1, progress.renewals)
		assert.Equal(t, 1, progress.records)
		assert.Empty(t, fanoutQueue.values)
		assert.Len(t, notificationQueue.values, 1)
		notification := notificationQueue.values[0].(domain.NotificationJob)
		assert.Equal(t, "event-1", notification.EventID)
		assert.Equal(t, "post-1", notification.PostID)
		assert.Equal(t, "author-1", notification.AuthorID)
		assert.Equal(t, []string{"user-1", "user-2"}, notification.RecipientIDs)
		assert.Equal(t, fixedNow.Format(time.RFC3339Nano), notification.CreatedAt)
	})

	t.Run("returns ErrLeaseHeld without processing followers", func(t *testing.T) {
		followers := &followerRepositoryStub{}
		progress := &progressRepositoryStub{acquired: false}
		useCase := NewProcessFanoutUseCase(followers, progress, &queueStub{}, &queueStub{}, time.Minute)

		err := useCase.Process(context.Background(), job, "owner-1")

		assert.Equal(t, ErrLeaseHeld, err)
		assert.Equal(t, 0, followers.calls)
	})

	t.Run("enqueues a continuation after the invocation limit", func(t *testing.T) {
		recipients := make([]string, pageSize)
		for i := range recipients { recipients[i] = "user" }
		followers := &followerRepositoryStub{page: domain.FollowerPage{RecipientIDs: recipients, NextCursor: "cursor-10"}}
		progress := &progressRepositoryStub{acquired: true}
		fanoutQueue, notificationQueue := &queueStub{}, &queueStub{}
		useCase := NewProcessFanoutUseCase(followers, progress, fanoutQueue, notificationQueue, time.Minute)
		useCase.now = func() time.Time { return fixedNow }

		err := useCase.Process(context.Background(), job, "owner-1")

		assert.NoError(t, err)
		assert.Equal(t, 10, followers.calls)
		assert.Len(t, notificationQueue.values, 20)
		assert.Equal(t, 1, progress.releases)
		assert.Equal(t, []any{domain.FanoutJob{EventID: "event-1", FanoutID: "event-1", JobID: "event-1#CURSOR#" + cursorHash("cursor-10"), PostID: "post-1", AuthorID: "author-1", Cursor: "cursor-10", BatchNumber: 1, CreatedAt: fixedNow.Format(time.RFC3339Nano)}}, fanoutQueue.values)
	})

	t.Run("returns the follower repository error", func(t *testing.T) {
		wantErr := errors.New("followers unavailable")
		useCase := NewProcessFanoutUseCase(&followerRepositoryStub{err: wantErr}, &progressRepositoryStub{acquired: true}, &queueStub{}, &queueStub{}, time.Minute)

		err := useCase.Process(context.Background(), job, "owner-1")

		assert.EqualError(t, err, wantErr.Error())
	})
}
