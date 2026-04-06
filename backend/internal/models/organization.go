package models

import (
	"time"

	"gorm.io/gorm"
)

// Organization represents a tenant organization in the system.
type Organization struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"not null;type:varchar(100)" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null;type:varchar(100)" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	OwnerID     uint           `gorm:"not null" json:"ownerId"`
	IsActive    bool           `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for Organization.
func (Organization) TableName() string {
	return "organizations"
}
