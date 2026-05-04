import { sanitizeErrorMessage } from "@/core/error-sanitizer"
import { config } from "@/core/config"

// Core-local types - avoids circular dependency with domains/
// domains/common re-exports these same shapes, but core/ cannot import domains/
export interface ApiResponse<T> {
  data: T
  error: string | null
  meta?: { page: number; limit: number; total: number }
}

interface TokenRefreshResponse {
  user: { id: string }
  accessToken: string
  refreshToken: string
}

// FINDING-11: Tokens are stored in localStorage for client-side auth state.
// This is a UI convenience - the backend enforces authentication and
// authorization on every endpoint. An XSS attack could steal these tokens,
// but backend rate limiting, short token TTLs, and refresh rotation mitigate risk.
// Migration to HttpOnly cookies would require backend changes.
const TOKEN_KEY = "vmx_access_token"
const REFRESH_TOKEN_KEY = "vmx_refresh_token"
const WORKSPACE_ID_KEY = "vmx_current_workspace_id"

export const getStoredAccessToken = (): string | null => {
  if (typeof window === "undefined") return null
  return localStorage.getItem(TOKEN_KEY)
}

export const getStoredRefreshToken = (): string | null => {
  if (typeof window === "undefined") return null
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export const storeTokens = (accessToken: string, refreshToken: string) => {
  localStorage.setItem(TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  // Set UX-hint cookie for SSR middleware route protection (not a security boundary)
  const secure = window.location.protocol === "https:" ? "; Secure" : ""
  document.cookie = `vmx_authenticated=1; path=/; SameSite=Lax; max-age=604800${secure}`
}

export const clearTokens = () => {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  // Clear UX-hint cookie so SSR middleware redirects to login
  document.cookie = "vmx_authenticated=; path=/; SameSite=Lax; max-age=0; Secure"
  document.cookie = "vmx_super_admin=; path=/; SameSite=Lax; max-age=0; Secure"
}

export const setSuperAdminHint = (isSuperAdmin: boolean) => {
  if (isSuperAdmin) {
    document.cookie = "vmx_super_admin=1; path=/; SameSite=Lax; max-age=604800"
  } else {
    document.cookie = "vmx_super_admin=; path=/; SameSite=Lax; max-age=0"
  }
}

export const getStoredWorkspaceId = (): string | null => {
  if (typeof window === "undefined") return null
  const v = localStorage.getItem(WORKSPACE_ID_KEY)
  return v || null
}

export const storeWorkspaceId = (workspaceId: string) => {
  localStorage.setItem(WORKSPACE_ID_KEY, workspaceId)
}

export const clearWorkspaceId = () => {
  localStorage.removeItem(WORKSPACE_ID_KEY)
}

// SEC-S4-002: Read CSRF token from cookie set by backend CSRF middleware
const getCsrfToken = (): string => {
  if (typeof document === "undefined") return ""
  const match = document.cookie.match(/(?:^|;\s*)_csrf_token=([^;]+)/)
  return match ? match[1] : ""
}

const CSRF_METHODS = ["POST", "PUT", "PATCH", "DELETE"]

let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

const attemptTokenRefresh = async (): Promise<boolean> => {
  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }
  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const refreshToken = getStoredRefreshToken()
      if (!refreshToken) return false

      const response = await fetch(`${config.apiBaseUrl}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken }),
      })

      if (!response.ok) return false

      const body = (await response.json()) as ApiResponse<TokenRefreshResponse>
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

export const fetchApi = async <T>(
  endpoint: string,
  options?: RequestInit & { skipAuth?: boolean }
): Promise<ApiResponse<T>> => {
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

    const workspaceId = getStoredWorkspaceId()
    if (workspaceId) {
      headers["X-Workspace-ID"] = String(workspaceId)
    }
  }

  // SEC-S4-002: Attach CSRF token for state-changing requests
  const method = (fetchOptions?.method ?? "GET").toUpperCase()
  if (CSRF_METHODS.includes(method)) {
    const csrfToken = getCsrfToken()
    if (csrfToken) {
      headers["X-CSRF-Token"] = csrfToken
    }
  }

  let response = await fetch(`${config.apiBaseUrl}${endpoint}`, {
    ...fetchOptions,
    credentials: "include",
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
      response = await fetch(`${config.apiBaseUrl}${endpoint}`, {
        ...fetchOptions,
        credentials: "include",
        headers,
      })
    } else {
      // Token refresh failed - session is expired
      clearTokens()
      clearWorkspaceId()
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("auth:session-expired"))
      }
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

  // Handle 204 No Content - no body to parse
  if (response.status === 204) {
    return { data: null as T, error: null }
  }

  let body: ApiResponse<T>
  try {
    body = (await response.json()) as ApiResponse<T>
  } catch {
    throw new Error(`API error: ${response.status}`)
  }

  if (!response.ok) {
    // SEC-S4-10: Sanitize raw API error messages before they reach UI consumers
    const rawMessage = body.error ?? `API error: ${response.status}`
    throw new Error(sanitizeErrorMessage(rawMessage))
  }

  return body
}
