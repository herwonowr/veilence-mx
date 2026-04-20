package id

import "github.com/google/uuid"

// New generates a new UUIDv7 string ID.
func New() string {
	return uuid.Must(uuid.NewV7()).String()
}
