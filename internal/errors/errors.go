package errors

import (
	"fmt"
)

// CustomError is the base type for all custom errors
type CustomError struct {
	message string
}

// Error implements the error interface
func (e *CustomError) Error() string {
	return e.message
}

// New creates a new CustomError
func New(message string) *CustomError {
	return &CustomError{message: message}
}

// Define specific custom errors
var (
	ErrNoChatMessages = New("not chat messages found")
)

// Is checks if the given error is of the specified custom error type
func Is(err error, target *CustomError) bool {
	customErr, ok := err.(*CustomError)
	if !ok {
		return false
	}
	return customErr.message == target.message
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
