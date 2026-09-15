package domain

import "time"

type TaskOverdueEvent struct {
	EventType     string    `json:"event_type"`
	EventID       string    `json:"event_id"`
	TaskID        string    `json:"task_id"`
	OwnerID       string    `json:"owner_id"`
	Description   string    `json:"description"`
	ExpiredAt     time.Time `json:"expired_at"`
	ExecutionTime time.Time `json:"execution_time"`
}
