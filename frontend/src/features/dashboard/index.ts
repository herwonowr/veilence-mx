"use client"

export {
  useDashboardStats,
  useChartData,
  useRecentReleases,
  useDashboardStalePackages,
  useDashboardSettings,
  dashboardKeys,
} from "@/features/dashboard/hooks/use-dashboard"
export { DashboardView } from "@/features/dashboard/ui/dashboard-view"
export { DashboardCharts } from "@/features/dashboard/ui/dashboard-charts"
export { PendingSuggestionsCard } from "@/features/dashboard/ui/pending-suggestions-card"
export { usePendingSuggestionsCount, pendingSuggestionsKeys } from "@/features/dashboard/hooks/use-pending-suggestions"
