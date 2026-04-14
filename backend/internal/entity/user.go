package entity

import "time"

// User represents an authenticated user of the system.
type User struct {
	ID             uint `json:"id"`
	Email          string `json:"email"`
	PasswordHash   string `json:"-"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	IsActive       bool `json:"isActive"`
	EmailVerified  bool `json:"emailVerified"`
	LastLoginAt    *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
