import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type {
  PlatformAuthSettings,
  UpdatePlatformAuthSettingsRequest,
  ListUsersResponse,
  PlatformUserDetail,
  UpdatePlatformUserRequest,
  PlatformAuditLog,
  PlatformAuditLogParams,
} from "@/domains/platform-admin/types/platform-admin.types"

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const platformAdminKeys = {
  all: ["platform-admin"] as const,
  authSettings: () => [...platformAdminKeys.all, "auth-settings"] as const,
  users: (params?: Record<string, string | number | undefined>) =>
    [...platformAdminKeys.all, "users", params] as const,
  user: (id: string) => [...platformAdminKeys.all, "user", id] as const,
  auditLogs: (params?: Record<string, string | number | undefined>) =>
    [...platformAdminKeys.all, "audit-logs", params] as const,
}

// ---------------------------------------------------------------------------
// Auth settings
// ---------------------------------------------------------------------------

/** GET /api/admin/auth-settings */
export const apiGetAuthSettings = async (): Promise<
  ApiResponse<PlatformAuthSettings>
> => fetchApi<PlatformAuthSettings>("/api/admin/auth-settings")

/** PUT /api/admin/auth-settings */
export const apiUpdateAuthSettings = async (
  req: UpdatePlatformAuthSettingsRequest
): Promise<ApiResponse<PlatformAuthSettings>> =>
  fetchApi<PlatformAuthSettings>("/api/admin/auth-settings", {
    method: "PUT",
    body: JSON.stringify(req),
  })

// ---------------------------------------------------------------------------
// User management
// ---------------------------------------------------------------------------

/** GET /api/admin/users */
export const apiGetPlatformUsers = async (params: {
  page?: number
  pageSize?: number
  search?: string
  sortBy?: string
  sortDir?: string
  status?: string
  role?: string
}): Promise<ApiResponse<ListUsersResponse>> => {
  const searchParams = new URLSearchParams()
  if (params.page !== undefined) searchParams.set("page", String(params.page))
  if (params.pageSize !== undefined)
    searchParams.set("limit", String(params.pageSize))
  if (params.search) searchParams.set("search", params.search)
  if (params.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params.sortDir) searchParams.set("sort_dir", params.sortDir)
  if (params.status) searchParams.set("status", params.status)
  if (params.role) searchParams.set("role", params.role)
  const query = searchParams.toString()
  return fetchApi<ListUsersResponse>(
    `/api/admin/users${query ? `?${query}` : ""}`
  )
}

/** GET /api/admin/users/{id} */
export const apiGetPlatformUser = async (
  id: string
): Promise<ApiResponse<PlatformUserDetail>> =>
  fetchApi<PlatformUserDetail>(`/api/admin/users/${id}`)

/** PUT /api/admin/users/{id} */
export const apiUpdatePlatformUser = async (
  id: string,
  req: UpdatePlatformUserRequest,
  confirmPassword: string
): Promise<ApiResponse<PlatformUserDetail>> =>
  fetchApi<PlatformUserDetail>(`/api/admin/users/${id}`, {
    method: "PUT",
    body: JSON.stringify({ ...req, confirmPassword }),
  })

// ---------------------------------------------------------------------------
// Audit logs
// ---------------------------------------------------------------------------

/** GET /api/admin/audit-logs */
export const apiGetPlatformAuditLogs = async (
  params: PlatformAuditLogParams
): Promise<ApiResponse<PlatformAuditLog[]>> => {
  const searchParams = new URLSearchParams()
  if (params.page !== undefined) searchParams.set("page", String(params.page))
  if (params.limit !== undefined) searchParams.set("limit", String(params.limit))
  if (params.action) searchParams.set("action", params.action)
  if (params.resource) searchParams.set("resource", params.resource)
  if (params.user_email) searchParams.set("user_email", params.user_email)
  if (params.workspace_name) searchParams.set("workspace_name", params.workspace_name)
  if (params.from_date) searchParams.set("from_date", params.from_date)
  if (params.to_date) searchParams.set("to_date", params.to_date)
  if (params.sort_by) searchParams.set("sort_by", params.sort_by)
  if (params.sort_dir) searchParams.set("sort_dir", params.sort_dir)
  const query = searchParams.toString()
  return fetchApi<PlatformAuditLog[]>(
    `/api/admin/audit-logs${query ? `?${query}` : ""}`
  )
}
