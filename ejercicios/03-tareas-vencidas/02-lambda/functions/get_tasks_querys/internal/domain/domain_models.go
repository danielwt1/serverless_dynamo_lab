package domain

import "time"

type TaskModel struct {
	TaskId      string
	OwnerId     string
	Description string
	CreatedAt   *time.Time
	Expire_at   *time.Time
	Status      string
}

type QueryResult struct {
	Task   []TaskModel
	Cursor string
}
