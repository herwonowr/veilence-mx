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
