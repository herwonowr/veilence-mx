export type Registry = "pypi" | "npm"
export type Classification = "benign" | "suspicious" | "malicious" | "baseline"
export type AnalyzerType = "api" | "cli" | "copilot"
export type AlertSeverity = "low" | "medium" | "high" | "critical"
export type AlertStatus = "new" | "acknowledged" | "resolved"
export type ReleaseStatus = "pending" | "diffing" | "analyzing" | "completed" | "error"

export interface Package {
  id: number
  name: string
  registry: Registry
  latestVersion: string
  description: string
  isCustom: boolean
  rank: number | null
  createdAt: string
  updatedAt: string
}

export interface Release {
  id: number
  packageId: number
  version: string
  publishedAt: string
  tarballUrl: string
  sha256: string
  status: ReleaseStatus
  errorMessage?: string
  createdAt: string
}

export interface Diff {
  id: number
  releaseId: number
  prevReleaseId: number
  diffContent: string
  fileChangesCount: number
  linesAdded: number
  linesRemoved: number
  createdAt: string
}

export interface Analysis {
  id: number
  diffId: number
  classification: Classification
  confidence: number
  reasoning: string
  modelUsed: string
  analyzerType: AnalyzerType
  createdAt: string
}

export interface Alert {
  id: number
  analysisId: number
  packageId: number
  releaseId?: number
  severity: AlertSeverity
  status: AlertStatus
  message: string
  packageName: string
  packageRegistry: string
  createdAt: string
  updatedAt: string
}

export interface DashboardStats {
  totalPackages: number
  totalReleases: number
  pendingAnalyses: number
  activeAlerts: number
  recentMalicious: number
}

export interface RecentRelease extends Release {
  packageName: string
  packageRegistry: string
  classification?: Classification
}

export interface ReleaseDetail extends Release {
  diff?: Diff
  analysis?: Analysis
  package?: Package
  isBaseline?: boolean
}

export interface PaginationMeta {
  page: number
  limit: number
  total: number
}

export interface ApiResponse<T> {
  data: T
  error: string | null
  meta?: PaginationMeta
}

export interface ChartData {
  releaseActivity: { date: string; releases: number }[]
  classifications: { classification: string; count: number }[]
  registries: { registry: string; count: number }[]
  alertsBySeverity: { severity: string; count: number }[]
  releaseStatuses: { status: string; count: number }[]
}

export interface QueueStats {
  pending: number
  processing: number
  completed: number
  failed: number
  dead: number
}

export interface QueueStatsResponse {
  diff: QueueStats
  analyze: QueueStats
}

export interface QueueJob {
  id: string
  type: string
  referenceId: number
  status: string
  attempts: number
  maxAttempts: number
  lastError?: string
  createdAt: number
  updatedAt: number
  nextRunAt: number
}

// ─── Auth & Enterprise Types ─────────────────────────────────────

export interface User {
  id: number
  email: string
  firstName: string
  lastName: string
  isActive: boolean
  emailVerified: boolean
  lastLoginAt: string | null
  createdAt: string
  updatedAt: string
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
}

export interface LoginResponse {
  user: User
  accessToken: string
  refreshToken: string
}

export interface Organization {
  id: number
  name: string
  slug: string
  description: string
  ownerId: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface OrgMember {
  id: number
  orgId: number
  userId: number
  roleId: number
  role: Role
  joinedAt: string
  email?: string
  firstName?: string
  lastName?: string
}

export interface Role {
  id: number
  orgId: number
  name: string
  description: string
  isSystem: boolean
  permissions?: Permission[]
}

export interface Permission {
  id: number
  resource: string
  action: string
}

export type APIKeyScope = "read" | "write" | "admin"

export interface ApiKeyInfo {
  id: number
  userId: number
  name: string
  keyPrefix: string
  scope: APIKeyScope
  lastUsedAt: string | null
  expiresAt: string | null
  isActive: boolean
  createdAt: string
}

export interface Session {
  id: number
  userId: number
  ipAddress: string
  userAgent: string
  createdAt: string
  lastActive: string
  expiresAt: string
  isCurrent: boolean
}

export interface Notification {
  id: number
  orgId: number
  userId: number
  channelId: number
  alertId?: number
  title: string
  message: string
  isRead: boolean
  sentAt: string
  createdAt: string
}

export interface AuditLog {
  id: number
  userId: number
  orgId: number
  action: string
  resource: string
  resourceId: number
  details: string
  ipAddress: string
  userAgent: string
  correlationId: string
  createdAt: string
}

export type NotificationChannelType = "email" | "slack" | "webhook"

export interface NotificationChannel {
  id: number
  orgId: number
  name: string
  type: NotificationChannelType
  config: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface NotificationRule {
  id: number
  orgId: number
  channelId: number
  severity: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

// ─── Alert Notes ──────────────────────────────────────────────

export interface AlertNote {
  id: number
  alertId: number
  userId: number
  userEmail: string
  content: string
  createdAt: string
  updatedAt: string
}

// ─── Bulk Import ──────────────────────────────────────────────

export interface BulkImportError {
  name: string
  error: string
}

export interface BulkImportResult {
  imported: number
  skipped: number
  errors: BulkImportError[]
}

// ─── Analysis History ─────────────────────────────────────────

export interface AnalysisHistoryEntry {
  releaseId: number
  version: string
  classification: Classification
  confidence: number
  reasoning: string
  modelUsed: string
  analyzerType: AnalyzerType
  analyzedAt: string
  publishedAt: string
}
