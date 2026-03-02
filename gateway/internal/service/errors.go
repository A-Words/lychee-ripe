package service

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrConflict           = errors.New("conflict")
	ErrServiceUnavailable = errors.New("service unavailable")
)
