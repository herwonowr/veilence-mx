import type {
  ApiResponse,
  Package,
  Release,
  ReleaseDetail,
  Alert,
  DashboardStats,
  RecentRelease,
  ChartData,
  QueueStatsResponse,
  QueueJob,
  User,
  LoginResponse,
  Organization,
  OrgMember,
  Role,
  Permission,
  ApiKeyInfo,
  AuditLog,
  Notification,
  NotificationChannel,
  NotificationRule,
} from "@/types"

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"

const TOKEN_KEY = "vmx_access_token"
const REFRESH_TOKEN_KEY = "vmx_refresh_token"
const ORG_ID_KEY = "vmx_current_org_id"

export function getStoredAccessToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(TOKEN_KEY)
}

export function getStoredRefreshToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function storeTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
}

export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

export function getStoredOrgId(): number | null {
  if (typeof window === "undefined") return null
  const v = localStorage.getItem(ORG_ID_KEY)
  return v ? parseInt(v, 10) : null
}

export function storeOrgId(orgId: number) {
  localStorage.setItem(ORG_ID_KEY, String(orgId))
}

export function clearOrgId() {
  localStorage.removeItem(ORG_ID_KEY)
}

let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

async function attemptTokenRefresh(): Promise<boolean> {
  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }
  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const refreshToken = getStoredRefreshToken()
      if (!refreshToken) return false

      const response = await fetch(`${API_BASE}/api/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken }),
      })

      if (!response.ok) return false

      const body = (await response.json()) as ApiResponse<LoginResponse>
      if (body.data?.accessToken && body.data?.refreshToken) {
        storeTokens(body.data.accessToken, body.data.refreshToken)
        return true
      }
      return false
    } catch {
      return false
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()
  return refreshPromise
}

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit & { skipAuth?: boolean }
): Promise<ApiResponse<T>> {
  const { skipAuth, ...fetchOptions } = options ?? {}

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(fetchOptions?.headers as Record<string, string>),
  }

  if (!skipAuth) {
    const token = getStoredAccessToken()
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    const orgId = getStoredOrgId()
    if (orgId) {
      headers["X-Org-ID"] = String(orgId)
    }
  }

  let response = await fetch(`${API_BASE}${endpoint}`, {
    ...fetchOptions,
    headers,
  })

  // Handle 401 by refreshing token and retrying
  if (response.status === 401 && !skipAuth) {
    const refreshed = await attemptTokenRefresh()
    if (refreshed) {
      const newToken = getStoredAccessToken()
      if (newToken) {
        headers["Authorization"] = `Bearer ${newToken}`
      }
      response = await fetch(`${API_BASE}${endpoint}`, {
        ...fetchOptions,
        headers,
      })
    }
  }

  // SEC-S3-006: Handle 429 rate limiting with Retry-After header
  if (response.status === 429) {
    const retryAfter = response.headers.get("Retry-After")
    const seconds = retryAfter ? parseInt(retryAfter, 10) : 60
    throw new Error(
      `Rate limited. Please try again in ${seconds} second${seconds !== 1 ? "s" : ""}.`
    )
  }

  const body = (await response.json()) as ApiResponse<T>

  if (!response.ok) {
    throw new Error(body.error ?? `API error: ${response.status}`)
  }

  return body
}

// ─── Auth API ────────────────────────────────────────────────

export async function apiLogin(
  email: string,
  password: string
): Promise<ApiResponse<LoginResponse>> {
  return fetchApi<LoginResponse>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
    skipAuth: true,
  })
}

export async function apiRegister(data: {
  email: string
  password: string
  firstName: string
  lastName: string
}): Promise<ApiResponse<LoginResponse>> {
  return fetchApi<LoginResponse>("/api/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
    skipAuth: true,
  })
}

export async function apiRefreshToken(
  refreshToken: string
): Promise<ApiResponse<LoginResponse>> {
  return fetchApi<LoginResponse>("/api/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refreshToken }),
    skipAuth: true,
  })
}

export async function apiLogout(
  refreshToken: string
): Promise<ApiResponse<null>> {
  return fetchApi<null>("/api/auth/logout", {
    method: "POST",
    body: JSON.stringify({ refreshToken }),
  })
}

export async function apiGetMe(): Promise<ApiResponse<User>> {
  return fetchApi<User>("/api/auth/me")
}

export async function apiForgotPassword(
  email: string
): Promise<ApiResponse<{ message: string }>> {
  return fetchApi<{ message: string }>("/api/auth/forgot-password", {
    method: "POST",
    body: JSON.stringify({ email }),
    skipAuth: true,
  })
}

export async function apiResetPassword(
  token: string,
  password: string
): Promise<ApiResponse<{ message: string }>> {
  return fetchApi<{ message: string }>("/api/auth/reset-password", {
    method: "POST",
    body: JSON.stringify({ token, password }),
    skipAuth: true,
  })
}

export async function apiUpdateProfile(data: {
  firstName: string
  lastName: string
}): Promise<ApiResponse<User>> {
  return fetchApi<User>("/api/auth/me", {
    method: "PUT",
    body: JSON.stringify(data),
  })
}

export async function apiChangePassword(data: {
  currentPassword: string
  newPassword: string
}): Promise<ApiResponse<{ message: string }>> {
  return fetchApi<{ message: string }>("/api/auth/change-password", {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function apiSendVerificationEmail(): Promise<
  ApiResponse<{ message: string }>
> {
  return fetchApi<{ message: string }>("/api/auth/send-verification", {
    method: "POST",
  })
}

// ─── API Keys ────────────────────────────────────────────────

export async function apiCreateApiKey(data: {
  name: string
  expiresAt?: string
}): Promise<ApiResponse<ApiKeyInfo & { apiKey: string }>> {
  return fetchApi<ApiKeyInfo & { apiKey: string }>("/api/auth/api-keys", {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function apiGetApiKeys(): Promise<ApiResponse<ApiKeyInfo[]>> {
  return fetchApi<ApiKeyInfo[]>("/api/auth/api-keys")
}

export async function apiDeleteApiKey(
  id: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/auth/api-keys/${id}`, { method: "DELETE" })
}

// ─── Organizations ────────────────────────────────────────────

export async function apiCreateOrg(data: {
  name: string
  slug: string
  description?: string
}): Promise<ApiResponse<Organization>> {
  return fetchApi<Organization>("/api/orgs", {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function apiGetOrgs(): Promise<ApiResponse<Organization[]>> {
  return fetchApi<Organization[]>("/api/orgs")
}

export async function apiGetOrg(
  id: number
): Promise<ApiResponse<Organization>> {
  return fetchApi<Organization>(`/api/orgs/${id}`)
}

export async function apiUpdateOrg(
  id: number,
  data: { name?: string; description?: string }
): Promise<ApiResponse<Organization>> {
  return fetchApi<Organization>(`/api/orgs/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })
}

export async function apiDeleteOrg(
  id: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/orgs/${id}`, { method: "DELETE" })
}

// ─── Members ──────────────────────────────────────────────────

export async function apiGetOrgMembers(
  orgId: number
): Promise<ApiResponse<OrgMember[]>> {
  return fetchApi<OrgMember[]>(`/api/orgs/${orgId}/members`)
}

export async function apiInviteMember(
  orgId: number,
  data: { email: string; roleId: number }
): Promise<ApiResponse<{ token: string }>> {
  return fetchApi<{ token: string }>(`/api/orgs/${orgId}/invitations`, {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export async function apiRemoveMember(
  orgId: number,
  userId: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/orgs/${orgId}/members/${userId}`, {
    method: "DELETE",
  })
}

export async function apiUpdateMemberRole(
  orgId: number,
  userId: number,
  roleId: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/orgs/${orgId}/members/${userId}/role`, {
    method: "PUT",
    body: JSON.stringify({ roleId }),
  })
}

// ─── Roles & Permissions ──────────────────────────────────────

export async function apiGetOrgRoles(
  orgId: number
): Promise<ApiResponse<Role[]>> {
  return fetchApi<Role[]>(`/api/orgs/${orgId}/roles`)
}

export async function apiGetPermissions(): Promise<ApiResponse<Permission[]>> {
  return fetchApi<Permission[]>("/api/permissions")
}

// ─── Audit Logs ───────────────────────────────────────────────

export async function apiGetAuditLogs(
  orgId: number,
  params?: {
    action?: string
    resource?: string
    from?: string
    to?: string
    page?: number
    limit?: number
  }
): Promise<ApiResponse<AuditLog[]>> {
  const searchParams = new URLSearchParams()
  if (params?.action) searchParams.set("action", params.action)
  if (params?.resource) searchParams.set("resource", params.resource)
  if (params?.from) searchParams.set("from", params.from)
  if (params?.to) searchParams.set("to", params.to)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  const query = searchParams.toString()
  return fetchApi<AuditLog[]>(
    `/api/orgs/${orgId}/audit-logs${query ? `?${query}` : ""}`
  )
}

// ─── Notification Channels ────────────────────────────────────

export async function apiListChannels(
  orgId: number
): Promise<ApiResponse<NotificationChannel[]>> {
  return fetchApi<NotificationChannel[]>(
    `/api/orgs/${orgId}/notification-channels`
  )
}

export async function apiCreateChannel(
  orgId: number,
  data: { name: string; type: string; config: string }
): Promise<ApiResponse<NotificationChannel>> {
  return fetchApi<NotificationChannel>(
    `/api/orgs/${orgId}/notification-channels`,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  )
}

export async function apiUpdateChannel(
  orgId: number,
  id: number,
  data: { name: string; config: string; isActive: boolean }
): Promise<ApiResponse<NotificationChannel>> {
  return fetchApi<NotificationChannel>(
    `/api/orgs/${orgId}/notification-channels/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(data),
    }
  )
}

export async function apiDeleteChannel(
  orgId: number,
  id: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/orgs/${orgId}/notification-channels/${id}`, {
    method: "DELETE",
  })
}

// ─── Notification Rules ──────────────────────────────────────

export async function apiListRules(
  orgId: number
): Promise<ApiResponse<NotificationRule[]>> {
  return fetchApi<NotificationRule[]>(
    `/api/orgs/${orgId}/notification-rules`
  )
}

export async function apiCreateRule(
  orgId: number,
  data: { channelId: number; severity: string }
): Promise<ApiResponse<NotificationRule>> {
  return fetchApi<NotificationRule>(
    `/api/orgs/${orgId}/notification-rules`,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  )
}

export async function apiDeleteRule(
  orgId: number,
  id: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/orgs/${orgId}/notification-rules/${id}`, {
    method: "DELETE",
  })
}

// ─── Existing API Functions ────────────────────────────────────

export async function getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
  return fetchApi<DashboardStats>("/api/dashboard/stats")
}

export async function getChartData(params?: {
  from?: string
  to?: string
}): Promise<ApiResponse<ChartData>> {
  const searchParams = new URLSearchParams()
  if (params?.from) searchParams.set("from", params.from)
  if (params?.to) searchParams.set("to", params.to)
  const query = searchParams.toString()
  return fetchApi<ChartData>(`/api/dashboard/charts${query ? `?${query}` : ""}`)
}

export async function getRecentReleases(params?: {
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
  search?: string
  registry?: string
  status?: string
  classification?: string
  latestPerPackage?: boolean
}): Promise<ApiResponse<RecentRelease[]>> {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  if (params?.search) searchParams.set("search", params.search)
  if (params?.registry) searchParams.set("registry", params.registry)
  if (params?.status) searchParams.set("status", params.status)
  if (params?.classification) searchParams.set("classification", params.classification)
  if (params?.latestPerPackage) searchParams.set("latest_per_package", "true")
  return fetchApi<RecentRelease[]>(
    `/api/dashboard/recent-releases?${searchParams.toString()}`
  )
}

export async function getPackages(params?: {
  registry?: string
  search?: string
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
}): Promise<ApiResponse<Package[]>> {
  const searchParams = new URLSearchParams()
  if (params?.registry) searchParams.set("registry", params.registry)
  if (params?.search) searchParams.set("search", params.search)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  return fetchApi<Package[]>(`/api/packages?${searchParams.toString()}`)
}

export async function getPackage(id: number): Promise<ApiResponse<Package>> {
  return fetchApi<Package>(`/api/packages/${id}`)
}

export async function createPackage(
  name: string,
  registry: string
): Promise<ApiResponse<Package>> {
  return fetchApi<Package>("/api/packages", {
    method: "POST",
    body: JSON.stringify({ name, registry }),
  })
}

export async function deletePackage(id: number): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/packages/${id}`, { method: "DELETE" })
}

export async function getPackageReleases(
  packageId: number,
  page = 1,
  limit = 20
): Promise<ApiResponse<Release[]>> {
  return fetchApi<Release[]>(
    `/api/packages/${packageId}/releases?page=${page}&limit=${limit}`
  )
}

export async function getRelease(
  id: number
): Promise<ApiResponse<ReleaseDetail>> {
  return fetchApi<ReleaseDetail>(`/api/releases/${id}`)
}

export async function getAlerts(params?: {
  severity?: string
  status?: string
  page?: number
  limit?: number
  sortBy?: string
  sortDir?: string
}): Promise<ApiResponse<Alert[]>> {
  const searchParams = new URLSearchParams()
  if (params?.severity) searchParams.set("severity", params.severity)
  if (params?.status) searchParams.set("status", params.status)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.sortBy) searchParams.set("sort_by", params.sortBy)
  if (params?.sortDir) searchParams.set("sort_dir", params.sortDir)
  return fetchApi<Alert[]>(`/api/alerts?${searchParams.toString()}`)
}

export async function updateAlertStatus(
  id: number,
  status: string
): Promise<ApiResponse<Alert>> {
  return fetchApi<Alert>(`/api/alerts/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  })
}

export async function getSettings(): Promise<
  ApiResponse<Record<string, string>>
> {
  return fetchApi<Record<string, string>>("/api/settings")
}

export async function updateSettings(
  settings: Record<string, string>
): Promise<ApiResponse<Record<string, string>>> {
  return fetchApi<Record<string, string>>("/api/settings", {
    method: "PUT",
    body: JSON.stringify(settings),
  })
}

export async function syncTopPackages(
  registry?: string
): Promise<ApiResponse<{ message: string }>> {
  const query = registry ? `?registry=${registry}` : ""
  return fetchApi<{ message: string }>(`/api/sync/top-packages${query}`, {
    method: "POST",
  })
}

export async function reanalyzeAll(): Promise<
  ApiResponse<{ message: string; queued: number; dead_retried: number }>
> {
  return fetchApi<{ message: string; queued: number; dead_retried: number }>(
    "/api/sync/reanalyze",
    { method: "POST" }
  )
}

export async function getQueueStats(): Promise<
  ApiResponse<QueueStatsResponse>
> {
  return fetchApi<QueueStatsResponse>("/api/queue/stats")
}

export async function getDeadJobs(
  type?: string
): Promise<ApiResponse<QueueJob[]>> {
  const query = type ? `?type=${type}` : ""
  return fetchApi<QueueJob[]>(`/api/queue/dead${query}`)
}

export async function retryDeadJobs(
  type?: string
): Promise<ApiResponse<{ message: string; count: number }>> {
  const query = type ? `?type=${type}` : ""
  return fetchApi<{ message: string; count: number }>(
    `/api/queue/retry-dead${query}`,
    { method: "POST" }
  )
}

// ─── Notifications ──────────────────────────────────────────────

export async function apiGetUnreadCount(): Promise<
  ApiResponse<{ count: number }>
> {
  return fetchApi<{ count: number }>("/api/notifications/unread-count")
}

export async function apiListNotifications(params?: {
  page?: number
  limit?: number
}): Promise<ApiResponse<Notification[]>> {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  const query = searchParams.toString()
  return fetchApi<Notification[]>(
    `/api/notifications${query ? `?${query}` : ""}`
  )
}

export async function apiMarkNotificationRead(
  id: number
): Promise<ApiResponse<null>> {
  return fetchApi<null>(`/api/notifications/${id}/read`, { method: "PUT" })
}

// TODO(S2-8): Backend endpoint PUT /api/notifications/read-all does not exist yet.
// Uncomment when the backend adds the read-all endpoint in Sprint 2.
// export async function apiMarkAllNotificationsRead(): Promise<
//   ApiResponse<null>
// > {
//   return fetchApi<null>("/api/notifications/read-all", { method: "PUT" })
// }
