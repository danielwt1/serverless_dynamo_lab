package in

import "context"

type FindExpiredTasks interface {
	Execute(ctx context.Context, executionTime string) error
}
