package models

import "time"

// Session represents an active user session with metadata for session management.
type Session struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"userId"`
	TokenHash  string    `gorm:"not null;uniqueIndex;type:varchar(255)" json:"-"`
	IPAddress  string    `gorm:"type:varchar(45)" json:"ipAddress"`
	UserAgent  string    `gorm:"type:varchar(512)" json:"userAgent"`
	CreatedAt  time.Time `json:"createdAt"`
	LastActive time.Time `gorm:"not null" json:"lastActive"`
	ExpiresAt  time.Time `gorm:"not null" json:"expiresAt"`
}

// TableName returns the table name for Session.
func (Session) TableName() string {
	return "sessions"
}
