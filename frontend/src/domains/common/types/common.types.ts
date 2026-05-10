// Re-export ApiResponse from core (source of truth for the HTTP layer shape)
export type { ApiResponse } from "@/core/http"

export type Ecosystem = "python" | "npm" | "go"

export const CLASSIFICATIONS = ["malicious", "suspicious", "benign", "baseline"] as const
export type Classification = (typeof CLASSIFICATIONS)[number]

export type AnalyzerType = "copilot" | "openai" | "anthropic" | "ollama"

export const ALERT_SEVERITIES = ["low", "medium", "high", "critical"] as const
export type AlertSeverity = (typeof ALERT_SEVERITIES)[number]

export const ALERT_STATUSES = ["new", "acknowledged", "resolved"] as const
export type AlertStatus = (typeof ALERT_STATUSES)[number]

// Backend statuses: pending, diffing, analyzing, completed, error.
// "in_progress" is a frontend-only grouping used in UI filters to represent diffing + analyzing.
export const RELEASE_STATUSES = ["pending", "diffing", "analyzing", "completed", "error", "in_progress"] as const
export type ReleaseStatus = (typeof RELEASE_STATUSES)[number]