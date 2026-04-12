import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { Alert, AlertNote } from "@/domains/alerts/types/alerts.types"

export const getAlerts = async (params?: {
  severity?: string
  status?: string
  search?: string
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
}): Promise<ApiResponse<Alert[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.severity) searchParams.set("severity", params.severity)
  if (params?.status) searchParams.set("status", params.status)
  if (params?.search) searchParams.set("search", params.search)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  return fetchApi<Alert[]>(`/api/alerts?${searchParams.toString()}`)
}

export const getAlert = async (id: number): Promise<ApiResponse<Alert>> =>
  fetchApi<Alert>(`/api/alerts/${id}`)

export const updateAlertStatus = async (
  id: number,
  status: string
): Promise<ApiResponse<Alert>> =>
  fetchApi<Alert>(`/api/alerts/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  })

export const getAlertNotes = async (
  alertId: number
): Promise<ApiResponse<AlertNote[]>> =>
  fetchApi<AlertNote[]>(`/api/alerts/${alertId}/notes`)

export const createAlertNote = async (
  alertId: number,
  content: string
): Promise<ApiResponse<AlertNote>> =>
  fetchApi<AlertNote>(`/api/alerts/${alertId}/notes`, {
    method: "POST",
    body: JSON.stringify({ content }),
  })
