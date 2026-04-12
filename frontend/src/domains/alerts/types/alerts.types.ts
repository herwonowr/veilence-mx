import type { AlertSeverity, AlertStatus } from "@/domains/common"

export interface Alert {
  id: number
  analysisId: number
  packageId: number
  releaseId?: number
  severity: AlertSeverity
  status: AlertStatus
  message: string
  packageName: string
  packageEcosystem: string
  createdAt: string
  updatedAt: string
}

export interface AlertNote {
  id: number
  alertId: number
  userId: number
  userEmail: string
  content: string
  createdAt: string
  updatedAt: string
}
