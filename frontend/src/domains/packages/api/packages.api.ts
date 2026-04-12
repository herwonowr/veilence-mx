import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { Package, Release, BulkImportResult, AnalysisHistoryEntry } from "@/domains/packages/types/packages.types"

export const getPackages = async (params?: {
  ecosystem?: string
  search?: string
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
}): Promise<ApiResponse<Package[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.ecosystem) searchParams.set("ecosystem", params.ecosystem)
  if (params?.search) searchParams.set("search", params.search)
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

export const getPackageReleases = async (
  packageId: number,
  page = 1,
  limit = 20
): Promise<ApiResponse<Release[]>> =>
  fetchApi<Release[]>(
    `/api/packages/${packageId}/releases?page=${page}&limit=${limit}`
  )

export const syncTopPackages = async (
  ecosystem?: string
): Promise<ApiResponse<{ message: string }>> => {
  const query = ecosystem ? `?ecosystem=${ecosystem}` : ""
  return fetchApi<{ message: string }>(`/api/sync/top-packages${query}`, {
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
