package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"list-active-users/internal/domain"
	"github.com/aws/aws-lambda-go/events"
)

type listActiveUsersStub struct { limit int32; cursor string; users []domain.User; next string; err error }
func (s *listActiveUsersStub) ListActiveUsers(_ context.Context, limit int32, cursor string) ([]domain.User, string, error) { s.limit, s.cursor = limit, cursor; return s.users, s.next, s.err }

func TestListActiveUsersHandler_List(t *testing.T) {
	tests := []struct { name, limit, cursor, message string; err error; wantStatus int; wantLimit int32 }{
		{name: "uses default limit", wantStatus: 200, wantLimit: 10},
		{name: "passes query parameters", limit: "25", cursor: "previous", wantStatus: 200, wantLimit: 25},
		{name: "rejects invalid limit", limit: "0", wantStatus: 400, message: domain.ErrInvalidLimit.Error()},
		{name: "rejects invalid cursor", err: domain.ErrInvalidCursor, wantStatus: 400, message: domain.ErrInvalidCursor.Error()},
		{name: "hides unexpected errors", err: errors.New("timeout"), wantStatus: 500, message: "could not list active users"},
	}
	for _, tt := range tests { t.Run(tt.name, func(t *testing.T) {
		service := &listActiveUsersStub{users: []domain.User{{UserID: "1"}}, next: "next", err: tt.err}
		response, err := NewListActiveUsersHandler(service).List(context.Background(), events.APIGatewayV2HTTPRequest{QueryStringParameters: map[string]string{"limit": tt.limit, "cursor": tt.cursor}})
		if err != nil { t.Fatalf("List() error = %v", err) }; if response.StatusCode != tt.wantStatus { t.Errorf("status = %d, want %d", response.StatusCode, tt.wantStatus) }
		if tt.wantStatus == 200 { var body listActiveUsersResponse; _ = json.Unmarshal([]byte(response.Body), &body); if len(body.Items) != 1 || body.NextCursor != "next" || service.limit != tt.wantLimit || service.cursor != tt.cursor { t.Errorf("body=%#v service=%#v", body, service) } } else { var body errorResponse; _ = json.Unmarshal([]byte(response.Body), &body); if body.Message != tt.message { t.Errorf("message = %q, want %q", body.Message, tt.message) } }
	}) }
}
