package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents an authenticated user of the system.
type User struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	Email         string         `gorm:"uniqueIndex;not null;type:varchar(255)" json:"email"`
	PasswordHash  string         `gorm:"type:varchar(255)" json:"-"`
	FirstName     string         `gorm:"type:varchar(100)" json:"firstName"`
	LastName      string         `gorm:"type:varchar(100)" json:"lastName"`
	IsActive      bool           `gorm:"not null;default:true" json:"isActive"`
	EmailVerified bool           `gorm:"not null;default:false" json:"emailVerified"`
	LastLoginAt   *time.Time     `json:"lastLoginAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for User.
func (User) TableName() string {
	return "users"
}

// HashPassword hashes the given plaintext password and stores it on the user.
func (u *User) HashPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword compares the given plaintext password against the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
