import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { DashboardStats, ChartData, RecentRelease } from "@/domains/dashboard/types/dashboard.types"

export const getDashboardStats = async (): Promise<ApiResponse<DashboardStats>> =>
  fetchApi<DashboardStats>("/api/dashboard/stats")

export const getChartData = async (params?: {
  from?: string
  to?: string
}): Promise<ApiResponse<ChartData>> => {
  const searchParams = new URLSearchParams()
  if (params?.from) searchParams.set("from", params.from)
  if (params?.to) searchParams.set("to", params.to)
  const query = searchParams.toString()
  return fetchApi<ChartData>(`/api/dashboard/charts${query ? `?${query}` : ""}`)
}

export const getRecentReleases = async (params?: {
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
  search?: string
  ecosystem?: string
  status?: string
  classification?: string
  latestPerPackage?: boolean
}): Promise<ApiResponse<RecentRelease[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  if (params?.search) searchParams.set("search", params.search)
  if (params?.ecosystem) searchParams.set("ecosystem", params.ecosystem)
  if (params?.status) searchParams.set("status", params.status)
  if (params?.classification) searchParams.set("classification", params.classification)
  if (params?.latestPerPackage) searchParams.set("latest_per_package", "true")
  return fetchApi<RecentRelease[]>(
    `/api/dashboard/recent-releases?${searchParams.toString()}`
  )
}
