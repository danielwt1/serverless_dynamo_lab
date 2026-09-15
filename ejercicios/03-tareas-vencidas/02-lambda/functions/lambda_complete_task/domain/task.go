package domain

import (
	"errors"
	"time"
)

const (
	PendingStatus   = "PENDING"
	CompletedStatus = "COMPLETED"
)

var (
	ErrInvalidID      = errors.New("ownerId y taskId son obligatorios")
	ErrTaskNotFound   = errors.New("tarea no encontrada")
	ErrTaskNotPending = errors.New("la tarea no está pendiente")
)

type Task struct {
	TaskID      string     `json:"task_id"`
	OwnerID     string     `json:"owner_id"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiredAt   time.Time  `json:"expired_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string     `json:"status"`
}
