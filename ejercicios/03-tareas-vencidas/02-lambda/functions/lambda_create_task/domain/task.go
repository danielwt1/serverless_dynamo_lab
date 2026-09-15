package domain

import (
	"errors"
	"time"
)

const PendingStatus = "PENDING"

var (
	ErrInvalidTask = errors.New("tarea inválida")
	ErrTaskExists  = errors.New("la tarea ya existe")
)

type Task struct {
	TaskID      string    `json:"task_id"`
	OwnerID     string    `json:"owner_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiredAt   time.Time `json:"expired_at"`
	Status      string    `json:"status"`
	ShardID     string    `json:"-"`
}
