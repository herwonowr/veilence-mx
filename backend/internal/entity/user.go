package entity

import "time"

// User represents an authenticated user of the system.
type User struct {
	ID             uint
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
