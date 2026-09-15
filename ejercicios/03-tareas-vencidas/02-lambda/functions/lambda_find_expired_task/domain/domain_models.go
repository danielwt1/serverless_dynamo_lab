package domain

import "time"

type TaskModel struct {
	TaskId      string     `dynamo:"task_id"`
	OwnerId     string     `dynamo:"owner_id"`
	Description string     `dynamo:"description"`
	CreatedAt   *time.Time `dynamo:"created_at"`
	Expire_at   *time.Time `dynamo:"expired_at"`
	Status      string     `dynamo:"status"`
}

type QueryResult struct {
	Task   []TaskModel
	Cursor string
}

//entity_type, status, active_run_id, execution_time, shard_count, last_error, updated_at.

type ProcessModel struct {
	EntityTpe     string `dynamo:"entity_type"`
	Status        string `dynamo:"status"`
	ExecutionTime string `dynamo:"execution_time"`
	ActiveRunId   string `dynamo:"active_run_id"`
	ShardId       string `dynamo:"shard_id"`
	LastError     string `dynamo:"last_error"`
	UpdatedAt     string `dynamo:"updated_at"`
}

// entity_type, run_id, shard_id, status, cursor, pages_completed, updated_at, ttl.
type ProcessShardModel struct {
	EntityTpe       string `dynamo:"entity_type"`
	Status          string `dynamo:"status"`
	RunId           string `dynamo:"run_id"`
	ShardId         string `dynamo:"shard_id"`
	Cursor          string `dynamo:"cursor"`
	Pages_completed int32  `dynamo:"pages_completed"`
	Updated_at      string `dynamo:"updated_at"`
	Ttl             int64  `dynamo:"ttl"`
}

type TaskOverdueEvent struct {
	EventId       string    `dynamo:"event_id" json:"event_id"`
	TaskId        string    `dynamo:"task_id" json:"task_id"`
	OwnerId       string    `dynamo:"owner_id" json:"owner_id"`
	Description   string    `dynamo:"description" json:"description"`
	ExpiredAt     time.Time `dynamo:"expired_at" json:"expired_at"`
	ExecutionTime time.Time `dynamo:"execution_time" json:"execution_time"`
}
