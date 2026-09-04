package domain

import "errors"

var (
	ErrInvalidLimit  = errors.New("limit must be between 1 and 100")
	ErrInvalidCursor = errors.New("invalid cursor")
)
