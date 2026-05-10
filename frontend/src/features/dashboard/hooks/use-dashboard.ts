"use client"

import {
  useQuery,
  keepPreviousData,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  getDashboardStats,
  getChartData,
  getRecentReleases,
} from "@/domains/dashboard"
import { getStalePackages } from "@/domains/packages"
import { getSettings } from "@/domains/settings"
import type { ApiResponse } from "@/domains/common"
import type { DashboardStats, ChartData, RecentRelease } from "@/domains/dashboard"
import type { StalePackage } from "@/domains/packages"

export const dashboardKeys = {
  all: ["dashboard"] as const,
  stats: () => [...dashboardKeys.all, "stats"] as const,
  charts: (params?: { from?: string; to?: string }) =>
    [...dashboardKeys.all, "charts", params] as const,
  recentReleases: (params?: Record<string, unknown>) =>
    [...dashboardKeys.all, "recent-releases", params] as const,
  stalePackages: (months?: number) =>
    [...dashboardKeys.all, "stale-packages", months] as const,
}

export const useDashboardStats = (
  options?: Partial<UseQueryOptions<ApiResponse<DashboardStats>>>
) =>
  useQuery({
    queryKey: dashboardKeys.stats(),
    queryFn: () => getDashboardStats(),
    staleTime: 30 * 1000,
    ...options,
  })

export const useChartData = (
  params?: { from?: string; to?: string },
  options?: Partial<UseQueryOptions<ApiResponse<ChartData>>>
) =>
  useQuery({
    queryKey: dashboardKeys.charts(params),
    queryFn: () => getChartData(params),
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
    ...options,
  })

export const useRecentReleases = (
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
) =>
  useQuery({
    queryKey: dashboardKeys.recentReleases(params as Record<string, unknown>),
    queryFn: () => getRecentReleases(params),
    staleTime: 30 * 1000,
    ...options,
  })

export const useDashboardStalePackages = (
  months = 6,
  options?: Partial<UseQueryOptions<ApiResponse<StalePackage[]>>>
) =>
  useQuery({
    queryKey: dashboardKeys.stalePackages(months),
    queryFn: () => getStalePackages({ months }),
    staleTime: 60 * 1000,
    ...options,
  })

export const useDashboardSettings = (
  options?: Partial<UseQueryOptions<ApiResponse<Record<string, string>>>>
) =>
  useQuery({
    queryKey: ["settings"],
    queryFn: () => getSettings(),
    staleTime: 60 * 1000,
    ...options,
  })
