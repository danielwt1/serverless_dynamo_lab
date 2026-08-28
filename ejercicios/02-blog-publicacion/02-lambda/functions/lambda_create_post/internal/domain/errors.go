package domain

import "errors"

var (
	ErrInvalidAuthorID    = errors.New("authorId is required")
	ErrInvalidDescription = errors.New("description is required")
	ErrInvalidStatus      = errors.New("status must be DRAFT or PUBLISHED")
	ErrPostAlreadyExists  = errors.New("post already exists")
)
