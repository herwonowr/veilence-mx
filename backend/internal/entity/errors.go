package entity

import "errors"

// Sentinel errors returned by domain repositories and services.
// Services should check these with errors.Is() instead of string matching.
var (
	// ErrNotFound indicates that the requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrForbidden indicates that the user does not have permission for this action.
	ErrForbidden = errors.New("forbidden")
	// ErrConflict indicates a uniqueness or state conflict (e.g. duplicate entry).
	ErrConflict = errors.New("conflict")
	// ErrValidation indicates a business-rule validation failure (e.g. invalid setting value).
	ErrValidation = errors.New("validation")
	// ErrSetupAlreadyCompleted indicates that the initial setup has already been completed.
	ErrSetupAlreadyCompleted = errors.New("setup already completed")
	// ErrSSOConfigExists indicates that an SSO config for this provider already exists.
	ErrSSOConfigExists = errors.New("an SSO configuration for this provider already exists")
)

// ValidationError is a validation failure with a user-facing message.
// It unwraps to ErrValidation so errors.Is(err, ErrValidation) works,
// but .Error() returns only the clean user-facing message.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}
