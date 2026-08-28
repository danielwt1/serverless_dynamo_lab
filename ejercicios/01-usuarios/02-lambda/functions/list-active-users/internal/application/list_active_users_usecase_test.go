package application

import (
	"context"
	"errors"
	"testing"
	"list-active-users/internal/domain"
)

type activeUsersStub struct { limit int32; cursor string; users []domain.User; next string; err error }
func (s *activeUsersStub) ListActiveUsers(_ context.Context, limit int32, cursor string) ([]domain.User, string, error) { s.limit, s.cursor = limit, cursor; return s.users, s.next, s.err }

func TestListActiveUsersUseCase_DelegatesToRepository(t *testing.T) {
	repository := &activeUsersStub{users: []domain.User{{UserID: "1"}}, next: "cursor"}
	users, cursor, err := NewListActiveUsersUseCase(repository).ListActiveUsers(context.Background(), 20, "previous")
	if err != nil || len(users) != 1 || cursor != "cursor" || repository.limit != 20 || repository.cursor != "previous" { t.Errorf("result users=%#v cursor=%q err=%v repository=%#v", users, cursor, err, repository) }
}

func TestListActiveUsersUseCase_ReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("DynamoDB unavailable")
	_, _, err := NewListActiveUsersUseCase(&activeUsersStub{err: wantErr}).ListActiveUsers(context.Background(), 1, "")
	if !errors.Is(err, wantErr) { t.Errorf("error = %v, want %v", err, wantErr) }
}
