package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrServerNotFound          = errors.New("server not found")
	ErrServerNameAlreadyExists = errors.New("server name already exists")
)

type ElasticSearchError struct {
	StatusCode int
	Type       string
	Reason     string
}

func (e *ElasticSearchError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.StatusCode, e.Type, e.Reason)
}

func NewElasticSearchError(statusCode int, typeReason string, reason string) error {
	return &ElasticSearchError{
		StatusCode: statusCode,
		Type:       typeReason,
		Reason:     reason,
	}
}
