package service

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrConflict           = errors.New("conflict")
	ErrNotFound           = errors.New("not found")
	ErrServiceUnavailable = errors.New("service unavailable")
)
