package entity

import "time"

// DigestOrgConfig holds the digest configuration for a single workspace.
type DigestOrgConfig struct {
	WorkspaceID uint
	Frequency   string
	Recipients  string
}

// DigestTopAlert is a summary of an alert for the digest.
type DigestTopAlert struct {
	ID          uint
	PackageName string
	Severity    string
	Message     string
	CreatedAt   time.Time
}
