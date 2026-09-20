package apierror

import (
	"errors"
	"net/http"
)

// Error is the stable, client-facing representation of an API failure.
type Error struct {
	Code    string
	Message string
	Status  int
	Cause   error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Wrap(err error, status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message, Cause: err}
}

func Public(err error) *Error {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}

	return New(http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}
