package domain

import "time"

type TaskModel struct {
	TaskId      string     `json:"task_id"`
	OwnerId     string     `json:"owner_id"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	Expire_at   *time.Time `json:"expired_at,omitempty"`
	Status      string     `json:"status"`
}

type QueryResult struct {
	Task   []TaskModel `json:"tasks"`
	Cursor string      `json:"cursor,omitempty"`
}
