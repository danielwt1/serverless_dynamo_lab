package out

import (
	"context"
	"lambda_find_expired_task/domain"
)

type TaskOverdueRecipient interface {
	PublishTaskOverdue(ctx context.Context, event domain.TaskOverdueEvent) error
}
