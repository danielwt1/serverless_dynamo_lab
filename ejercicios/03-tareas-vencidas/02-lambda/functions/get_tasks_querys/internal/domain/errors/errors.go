package errors

import "errors"

var (
	InvalidCursor = errors.New("invalid cursor")
	DynamoError   = errors.New("dynamodb Error")
)
