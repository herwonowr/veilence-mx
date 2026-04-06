package models

import (
	"time"

	"gorm.io/gorm"
)

// APIKey represents a long-lived API key for programmatic access.
type APIKey struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	UserID     uint           `gorm:"not null;index" json:"userId"`
	Name       string         `gorm:"not null;type:varchar(100)" json:"name"`
	KeyHash    string         `gorm:"uniqueIndex;not null;type:varchar(255)" json:"-"`
	KeyPrefix  string         `gorm:"not null;type:varchar(10)" json:"keyPrefix"`
	Scope      string         `gorm:"not null;type:varchar(20);default:'read'" json:"scope"`
	LastUsedAt *time.Time     `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time     `json:"expiresAt,omitempty"`
	IsActive   bool           `gorm:"not null;default:true" json:"isActive"`
	CreatedAt  time.Time      `json:"createdAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for APIKey.
func (APIKey) TableName() string {
	return "api_keys"
}
