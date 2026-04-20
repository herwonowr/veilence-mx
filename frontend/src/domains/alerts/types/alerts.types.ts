import type { AlertSeverity, AlertStatus } from "@/domains/common"

export interface Alert {
  id: string
  workspaceId: string
  analysisId: string
  packageId: string
  releaseId: string
  severity: AlertSeverity
  status: AlertStatus
  message: string
  packageName: string
  packageEcosystem: string
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
