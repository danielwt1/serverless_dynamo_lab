package domain

import "errors"

var (
	UserNotFound                = errors.New("user not found")
	UserNotFoundOrPassWordWrong = errors.New("user not found or password wrong")
)
