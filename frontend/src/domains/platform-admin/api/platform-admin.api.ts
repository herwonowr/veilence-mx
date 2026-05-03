import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type {
  PlatformAuthSettings,
  UpdatePlatformAuthSettingsRequest,
  ListUsersResponse,
  PlatformUserDetail,
  UpdatePlatformUserRequest,
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
  sort?: string
  order?: string
}): Promise<ApiResponse<ListUsersResponse>> => {
  const searchParams = new URLSearchParams()
  if (params.page !== undefined) searchParams.set("page", String(params.page))
  if (params.pageSize !== undefined)
    searchParams.set("pageSize", String(params.pageSize))
  if (params.search) searchParams.set("search", params.search)
  if (params.sort) searchParams.set("sort", params.sort)
  if (params.order) searchParams.set("order", params.order)
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
    headers: { "X-Confirm-Password": confirmPassword },
    body: JSON.stringify(req),
  })
