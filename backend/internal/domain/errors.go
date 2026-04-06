package domain

import "errors"

// Sentinel errors returned by domain repositories and services.
// Services should check these with errors.Is() instead of string matching.
var (
	// ErrNotFound indicates that the requested entity does not exist.
	ErrNotFound = errors.New("not found")
)
