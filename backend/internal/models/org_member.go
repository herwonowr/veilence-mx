package models

import (
	"time"
)

// OrgMember represents a user's membership in an organization with a specific role.
type OrgMember struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	OrgID     uint      `gorm:"not null;uniqueIndex:idx_org_user" json:"orgId"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_org_user" json:"userId"`
	RoleID    uint      `gorm:"not null" json:"roleId"`
	Role      Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	JoinedAt  time.Time `gorm:"not null" json:"joinedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName returns the table name for OrgMember.
func (OrgMember) TableName() string {
	return "org_members"
}
