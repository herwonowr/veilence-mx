package entity

import "time"

// User represents an authenticated user of the system.
type User struct {
	ID             string
	Email          string
	PasswordHash   string
	FirstName      string
	LastName       string
	IsActive       bool
	EmailVerified  bool
	LastLoginAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
