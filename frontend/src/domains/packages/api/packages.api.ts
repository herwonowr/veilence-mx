import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { Package, Release, BulkImportResult, AnalysisHistoryEntry, StalePackage } from "@/domains/packages/types/packages.types"

export const getPackages = async (params?: {
  ecosystem?: string
  search?: string
  status?: string
  source?: string
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
}): Promise<ApiResponse<Package[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.ecosystem) searchParams.set("ecosystem", params.ecosystem)
  if (params?.search) searchParams.set("search", params.search)
  if (params?.status) searchParams.set("status", params.status)
  if (params?.source) searchParams.set("source", params.source)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  return fetchApi<Package[]>(`/api/packages?${searchParams.toString()}`)
}

export const getPackage = async (id: number): Promise<ApiResponse<Package>> =>
  fetchApi<Package>(`/api/packages/${id}`)

export const createPackage = async (
  name: string,
  ecosystem: string
): Promise<ApiResponse<Package>> =>
  fetchApi<Package>("/api/packages", {
    method: "POST",
    body: JSON.stringify({ name, ecosystem }),
  })

export const deletePackage = async (id: number): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/packages/${id}`, { method: "DELETE" })

export const blockPackage = async (
  id: number,
  reason?: string
): Promise<ApiResponse<Package>> =>
  fetchApi<Package>(`/api/packages/${id}/block`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  })

export const unblockPackage = async (
  id: number
): Promise<ApiResponse<Package>> =>
  fetchApi<Package>(`/api/packages/${id}/unblock`, {
    method: "POST",
  })

export const getPackageReleases = async (
  packageId: number,
  page = 1,
  limit = 20
): Promise<ApiResponse<Release[]>> =>
  fetchApi<Release[]>(
    `/api/packages/${packageId}/releases?page=${page}&limit=${limit}`
  )

export const discoverPackages = async (
  ecosystem?: string
): Promise<ApiResponse<{ message: string }>> => {
  const query = ecosystem ? `?ecosystem=${ecosystem}` : ""
  return fetchApi<{ message: string }>(`/api/sync/discover${query}`, {
    method: "POST",
  })
}

export const bulkImportPackages = async (
  format: "requirements_txt" | "package_json" | "list",
  content: string
): Promise<ApiResponse<BulkImportResult>> =>
  fetchApi<BulkImportResult>("/api/packages/bulk-import", {
    method: "POST",
    body: JSON.stringify({ format, content }),
  })

export const getAnalysisHistory = async (
  packageId: number
): Promise<ApiResponse<AnalysisHistoryEntry[]>> =>
  fetchApi<AnalysisHistoryEntry[]>(
    `/api/packages/${packageId}/analysis-history`
  )

export const getPackageSuggestions = async (params?: {
  page?: number
  limit?: number
  search?: string
  ecosystem?: string
  sortBy?: string
  sortDir?: string
}): Promise<ApiResponse<Package[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.search) searchParams.set("search", params.search)
  if (params?.ecosystem) searchParams.set("ecosystem", params.ecosystem)
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  return fetchApi<Package[]>(`/api/packages/suggestions?${searchParams.toString()}`)
}

export const approvePackage = async (id: number): Promise<ApiResponse<Package>> =>
  fetchApi<Package>(`/api/packages/${id}/approve`, { method: "POST" })

export const rejectPackage = async (id: number): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/packages/${id}/reject`, { method: "POST" })

export const bulkApprovePackages = async (
  params: { packageIds: number[] } | { ecosystem: string } | { approveAll: true }
): Promise<ApiResponse<{ approved: number }>> =>
  fetchApi<{ approved: number }>("/api/packages/bulk-approve", {
    method: "POST",
    body: JSON.stringify(params),
  })

export const getStalePackages = async (months?: number): Promise<ApiResponse<StalePackage[]>> => {
  const query = months ? `?months=${months}` : ""
  return fetchApi<StalePackage[]>(`/api/packages/stale${query}`)
}
