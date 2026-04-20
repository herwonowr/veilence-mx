package entity

import "time"

// DigestOrgConfig holds the digest configuration for a single workspace.
type DigestOrgConfig struct {
	WorkspaceID string
	Frequency   string
	Recipients  string
}

// DigestTopAlert is a summary of an alert for the digest.
type DigestTopAlert struct {
	ID          string
	PackageName string
	Severity    string
	Message     string
	CreatedAt   time.Time
}
