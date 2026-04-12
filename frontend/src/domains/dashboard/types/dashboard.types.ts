import type { Classification, ReleaseStatus } from "@/domains/common"

export interface DashboardStats {
  totalPackages: number
  totalReleases: number
  pendingAnalyses: number
  activeAlerts: number
  recentMalicious: number
}

export interface ChartData {
  releaseActivity: { date: string; releases: number }[]
  classifications: { classification: string; count: number }[]
  ecosystems: { ecosystem: string; count: number }[]
  alertsBySeverity: { severity: string; count: number }[]
  releaseStatuses: { status: string; count: number }[]
}

export interface RecentRelease {
  id: number
  packageId: number
  version: string
  publishedAt: string
  tarballUrl: string
  sha256: string
  status: ReleaseStatus
  errorMessage?: string
  createdAt: string
  packageName: string
  packageEcosystem: string
  classification?: Classification
}
