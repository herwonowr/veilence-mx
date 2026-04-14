// Package apperror provides structured application error types with HTTP status
// code mapping for consistent API error responses.
package v1

import (
	"fmt"
	"net/http"
)

// Error is the base application error type. It implements the error interface
// and carries an HTTP status code and machine-readable error code.
type Error struct {
	// HTTPStatus is the HTTP status code to return (e.g. 400, 404, 500).
	HTTPStatus int `json:"-"`
	// Code is a machine-readable error code (e.g. "not_found", "validation_error").
	Code string `json:"code"`
	// Message is a human-readable error message.
	Message string `json:"message"`
	// cause is the underlying error, if any.
	cause error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *Error) Unwrap() error {
	return e.cause
}

// WithCause returns a copy of the error with the given underlying cause.
func (e *Error) WithCause(err error) *Error {
	return &Error{
		HTTPStatus: e.HTTPStatus,
		Code:       e.Code,
		Message:    e.Message,
		cause:      err,
	}
}

// --- Constructors ---

// NotFound creates a 404 Not Found error.
func NotFound(resource string) *Error {
	return &Error{
		HTTPStatus: http.StatusNotFound,
		Code:       "not_found",
		Message:    resource + " not found",
	}
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(msg string) *Error {
	return &Error{
		HTTPStatus: http.StatusUnauthorized,
		Code:       "unauthorized",
		Message:    msg,
	}
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(msg string) *Error {
	return &Error{
		HTTPStatus: http.StatusForbidden,
		Code:       "forbidden",
		Message:    msg,
	}
}

// Validation creates a 400 Bad Request error for validation failures.
func Validation(msg string) *Error {
	return &Error{
		HTTPStatus: http.StatusBadRequest,
		Code:       "validation_error",
		Message:    msg,
	}
}

// Conflict creates a 409 Conflict error.
func Conflict(msg string) *Error {
	return &Error{
		HTTPStatus: http.StatusConflict,
		Code:       "conflict",
		Message:    msg,
	}
}

// Internal creates a 500 Internal Server Error.
func Internal(msg string) *Error {
	return &Error{
		HTTPStatus: http.StatusInternalServerError,
		Code:       "internal_error",
		Message:    msg,
	}
}

// BadRequest creates a 400 Bad Request error.
func BadRequest(msg string) *Error {
	return &Error{
		HTTPStatus: http.StatusBadRequest,
		Code:       "bad_request",
		Message:    msg,
	}
}
