import {
  useQuery,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  getDashboardStats,
  getChartData,
  getRecentReleases,
} from "@/lib/api-client"
import type { ApiResponse, DashboardStats, ChartData, RecentRelease } from "@/types"

export const dashboardKeys = {
  all: ["dashboard"] as const,
  stats: () => [...dashboardKeys.all, "stats"] as const,
  charts: (params?: { from?: string; to?: string }) =>
    [...dashboardKeys.all, "charts", params] as const,
  recentReleases: (params?: Record<string, unknown>) =>
    [...dashboardKeys.all, "recent-releases", params] as const,
}

export function useDashboardStats(
  options?: Partial<UseQueryOptions<ApiResponse<DashboardStats>>>
) {
  return useQuery({
    queryKey: dashboardKeys.stats(),
    queryFn: () => getDashboardStats(),
    staleTime: 30 * 1000, // stale-while-revalidate: 30s
    ...options,
  })
}

export function useChartData(
  params?: { from?: string; to?: string },
  options?: Partial<UseQueryOptions<ApiResponse<ChartData>>>
) {
  return useQuery({
    queryKey: dashboardKeys.charts(params),
    queryFn: () => getChartData(params),
    staleTime: 30 * 1000,
    ...options,
  })
}

export function useRecentReleases(
  params?: {
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
    search?: string
    ecosystem?: string
    status?: string
    classification?: string
    latestPerPackage?: boolean
  },
  options?: Partial<UseQueryOptions<ApiResponse<RecentRelease[]>>>
) {
  return useQuery({
    queryKey: dashboardKeys.recentReleases(params as Record<string, unknown>),
    queryFn: () => getRecentReleases(params),
    staleTime: 30 * 1000,
    ...options,
  })
}
