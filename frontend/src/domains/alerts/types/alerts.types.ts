import type { AlertSeverity, AlertStatus } from "@/domains/common"

// The backend has two response shapes: AlertResponse (base) and AlertWithPackageResponse.
// List and detail endpoints return AlertWithPackageResponse (with packageName/packageEcosystem).
// The update-status endpoint returns base AlertResponse (without packageName/packageEcosystem).
// We use a single type with optional package fields for pragmatic FE consumption.
export interface Alert {
  id: string
  workspaceId: string
  analysisId: string | null
  packageId: string
  releaseId: string | null
  severity: AlertSeverity
  status: AlertStatus
  message: string
  packageName?: string
  packageEcosystem?: string
  createdAt: string
  updatedAt: string
}

export interface AlertNote {
  id: string
  workspaceId: string
  alertId: string
  userId: string
  userEmail: string
  content: string
  createdAt: string
  updatedAt: string
}
