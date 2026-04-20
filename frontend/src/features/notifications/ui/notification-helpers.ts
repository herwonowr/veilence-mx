import type { ComponentType } from "react"
import {
  Package,
  AlertTriangle,
  ShieldAlert,
  ShieldX,
  Bug,
  Search,
  Trash2,
} from "lucide-react"
import type { Notification } from "@/domains/notifications"

// ---------------------------------------------------------------------------
// Severity
// ---------------------------------------------------------------------------

export type SeverityLevel = "critical" | "high" | "medium" | "info"

const VALID_SEVERITIES: ReadonlyArray<SeverityLevel> = [
  "critical",
  "high",
  "medium",
  "info",
]

export const classifySeverity = (notification: Notification): SeverityLevel => {
  const mapped = notification.severity === "low" ? "info" : notification.severity
  return VALID_SEVERITIES.includes(mapped as SeverityLevel)
    ? (mapped as SeverityLevel)
    : "info"
}

export const SEVERITY_STYLES: Record<SeverityLevel, string> = {
  critical:
    "bg-red-500/10 text-red-700 dark:bg-red-500/20 dark:text-red-400 border-red-200 dark:border-red-800",
  high:
    "bg-orange-500/10 text-orange-700 dark:bg-orange-500/20 dark:text-orange-400 border-orange-200 dark:border-orange-800",
  medium:
    "bg-amber-500/10 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400 border-amber-200 dark:border-amber-800",
  info:
    "bg-blue-500/10 text-blue-700 dark:bg-blue-500/20 dark:text-blue-400 border-blue-200 dark:border-blue-800",
}

// ---------------------------------------------------------------------------
// Event type icon mapping
// ---------------------------------------------------------------------------

export type EventTypeConfig = {
  icon: ComponentType<{ className?: string }>
  bgClass: string
  textClass: string
}

const EVENT_TYPE_MAP: Record<string, EventTypeConfig> = {
  "discovery.packages_added": {
    icon: Package,
    bgClass: "bg-blue-500/10 dark:bg-blue-500/20",
    textClass: "text-blue-600 dark:text-blue-400",
  },
  "alert.created.malicious": {
    icon: ShieldX,
    bgClass: "bg-red-500/10 dark:bg-red-500/20",
    textClass: "text-red-600 dark:text-red-400",
  },
  "alert.created.suspicious": {
    icon: ShieldAlert,
    bgClass: "bg-orange-500/10 dark:bg-orange-500/20",
    textClass: "text-orange-600 dark:text-orange-400",
  },
  "analysis.error": {
    icon: AlertTriangle,
    bgClass: "bg-amber-500/10 dark:bg-amber-500/20",
    textClass: "text-amber-600 dark:text-amber-400",
  },
  "diff.error": {
    icon: Bug,
    bgClass: "bg-amber-500/10 dark:bg-amber-500/20",
    textClass: "text-amber-600 dark:text-amber-400",
  },
  "packages.stale_removed": {
    icon: Trash2,
    bgClass: "bg-gray-500/10 dark:bg-gray-500/20",
    textClass: "text-gray-600 dark:text-gray-400",
  },
}

const DEFAULT_EVENT_TYPE: EventTypeConfig = {
  icon: Search,
  bgClass: "bg-muted",
  textClass: "text-muted-foreground",
}

export const getEventTypeConfig = (eventType: string): EventTypeConfig =>
  EVENT_TYPE_MAP[eventType] ?? DEFAULT_EVENT_TYPE

// ---------------------------------------------------------------------------
// Relative timestamp
// ---------------------------------------------------------------------------

export const formatTimeAgo = (dateStr: string): string => {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const diffSec = Math.floor(diffMs / 1000)

  if (diffSec < 60) return "just now"
  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHr = Math.floor(diffMin / 60)
  if (diffHr < 24) return `${diffHr}h ago`
  const diffDay = Math.floor(diffHr / 24)
  if (diffDay < 30) return `${diffDay}d ago`
  return new Date(dateStr).toLocaleDateString()
}

// ---------------------------------------------------------------------------
// Event type human-readable labels
// ---------------------------------------------------------------------------

export const EVENT_TYPE_LABELS: Record<string, string> = {
  "discovery.packages_added": "Package Discovery",
  "alert.created.malicious": "Malicious Alert",
  "alert.created.suspicious": "Suspicious Alert",
  "analysis.error": "Analysis Error",
  "diff.error": "Diff Error",
  "packages.stale_removed": "Package Removed",
}

// ---------------------------------------------------------------------------
// Click-through navigation
// ---------------------------------------------------------------------------

export const getNotificationLink = (notification: Notification): string => {
  switch (notification.referenceType) {
    case "alert":
      return notification.referenceId
        ? `/alerts/${notification.referenceId}`
        : "/alerts"
    case "release":
      return notification.referenceId
        ? `/releases/${notification.referenceId}`
        : "/releases"
    case "package":
      return notification.referenceId
        ? `/packages/${notification.referenceId}`
        : "/packages"
    default:
      return "/dashboard"
  }
}
