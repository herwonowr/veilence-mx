// Re-export ApiResponse from core (source of truth for the HTTP layer shape)
export type { ApiResponse } from "@/core/http"

export type Ecosystem = "python" | "npm"
export type Classification = "benign" | "suspicious" | "malicious" | "baseline"
export type AnalyzerType = "copilot" | "openai" | "anthropic" | "ollama"
export type AlertSeverity = "low" | "medium" | "high" | "critical"
export type AlertStatus = "new" | "acknowledged" | "resolved"
export type ReleaseStatus = "pending" | "diffing" | "analyzing" | "completed" | "error" | "in_progress"
export interface PaginationMeta {
  page: number
  limit: number
  total: number
}


