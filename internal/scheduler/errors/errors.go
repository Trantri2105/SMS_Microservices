package apperrors

import (
	"errors"
)

var (
	ErrServerNotFound          = errors.New("server not found")
	ErrServerNameAlreadyExists = errors.New("server name already exists")
)
