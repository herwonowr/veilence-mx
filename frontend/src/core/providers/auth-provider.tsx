"use client"

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
// Core-local interfaces - auth-provider needs User/Org shapes but core/ cannot import domains/
// These types are structurally identical to domains/auth and domains/admin equivalents.
interface User {
  id: number
  email: string
  firstName: string
  lastName: string
  isActive: boolean
  emailVerified: boolean
  lastLoginAt: string | null
  createdAt: string
  updatedAt: string
}

interface Organization {
  id: number
  name: string
  slug: string
  description: string
  ownerId: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

interface LoginResponse {
  user: User
  accessToken: string
  refreshToken: string
}

import { sanitizeErrorMessage } from "@/core/error-sanitizer"
import {
  getStoredAccessToken,
  getStoredRefreshToken,
  storeTokens,
  clearTokens,
  getStoredOrgId,
  storeOrgId,
  clearOrgId,
  fetchApi,
} from "@/core/http"

// SEC-S4-10: Inactivity timeout constants (milliseconds)
const INACTIVITY_TIMEOUT_MS = 30 * 60 * 1000 // 30 minutes
const INACTIVITY_WARNING_MS = 25 * 60 * 1000  // 25 minutes (warn 5 min before logout)
const ACTIVITY_EVENTS: ReadonlyArray<keyof WindowEventMap> = [
  "mousemove",
  "keydown",
  "touchstart",
  "scroll",
  "click",
]

interface AuthContextValue {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  currentOrg: Organization | null
  organizations: Organization[]
  orgsLoading: boolean
  orgsError: string | null
  login: (email: string, password: string) => Promise<void>
  register: (data: {
    email: string
    password: string
    firstName: string
    lastName: string
  }) => Promise<void>
  logout: () => Promise<void>
  setCurrentOrg: (org: Organization) => void
  refreshUser: () => Promise<void>
  refreshOrgs: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return ctx
}

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const queryClient = useQueryClient()
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [currentOrg, setCurrentOrgState] = useState<Organization | null>(null)
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [orgsLoading, setOrgsLoading] = useState(false)
  const [orgsError, setOrgsError] = useState<string | null>(null)
  const isAuthenticated = !!user

  const refreshUser = useCallback(async () => {
    try {
      const { data } = await fetchApi<User>("/api/auth/me")
      setUser(data)
    } catch {
      setUser(null)
      clearTokens()
    }
  }, [])

  const refreshOrgs = useCallback(async () => {
    setOrgsLoading(true)
    setOrgsError(null)
    try {
      const { data } = await fetchApi<Organization[]>("/api/orgs")
      setOrganizations(data ?? [])

      // Try to restore current org from localStorage
      const storedOrgId = getStoredOrgId()
      if (storedOrgId && data) {
        const found = data.find((o) => o.id === storedOrgId)
        if (found) {
          setCurrentOrgState(found)
        } else if (data.length > 0) {
          setCurrentOrgState(data[0])
          storeOrgId(data[0].id)
        }
      } else if (data && data.length > 0) {
        setCurrentOrgState(data[0])
        storeOrgId(data[0].id)
      }
    } catch (err) {
      setOrganizations([])
      const message = sanitizeErrorMessage(err, "Failed to load organizations")
      setOrgsError(message)
      toast.error(message)
    } finally {
      setOrgsLoading(false)
    }
  }, [])

  // Initialize auth state from localStorage (must run after mount to avoid state updates before mount)
  useEffect(() => {
    const token = getStoredAccessToken()
    if (token) {
      Promise.all([refreshUser(), refreshOrgs()]).finally(() => {
        setIsLoading(false)
      })
    } else {
      setIsLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only run once on mount
  }, [])

  // Auto-refresh token before expiry (refresh every 10 minutes)
  useEffect(() => {
    if (!isAuthenticated) return

    const interval = setInterval(
      async () => {
        const refreshToken = getStoredRefreshToken()
        if (!refreshToken) return

        try {
          const { data } = await fetchApi<LoginResponse>("/api/auth/refresh", {
            method: "POST",
            body: JSON.stringify({ refreshToken }),
            skipAuth: true,
          })
          if (data?.accessToken && data?.refreshToken) {
            storeTokens(data.accessToken, data.refreshToken)
          }
        } catch {
          // Proactive refresh failed - session likely expired
          if (typeof window !== "undefined") {
            window.dispatchEvent(new CustomEvent("auth:session-expired"))
          }
        }
      },
      10 * 60 * 1000
    )

    return () => clearInterval(interval)
  }, [isAuthenticated])

  const login = useCallback(
    async (email: string, password: string) => {
      const { data } = await fetchApi<LoginResponse>("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
        skipAuth: true,
      })
      if (!data) {
        throw new Error("Login failed: no data received")
      }
      storeTokens(data.accessToken, data.refreshToken)
      setUser(data.user)
      await refreshOrgs()
    },
    [refreshOrgs]
  )

  const register = useCallback(
    async (params: {
      email: string
      password: string
      firstName: string
      lastName: string
    }) => {
      const { data } = await fetchApi<LoginResponse>("/api/auth/register", {
        method: "POST",
        body: JSON.stringify(params),
        skipAuth: true,
      })
      if (!data) {
        throw new Error("Registration failed: no data received")
      }
      storeTokens(data.accessToken, data.refreshToken)
      setUser(data.user)
      await refreshOrgs()
    },
    [refreshOrgs]
  )

  const logout = useCallback(async () => {
    const refreshToken = getStoredRefreshToken()
    if (refreshToken) {
      try {
        await fetchApi<null>("/api/auth/logout", {
          method: "POST",
          body: JSON.stringify({ refreshToken }),
        })
      } catch {
        // Logout API failure is not critical
      }
    }
    clearTokens()
    clearOrgId()
    setUser(null)
    setCurrentOrgState(null)
    setOrganizations([])
    // SEC-S3-002: Clear React Query cache to prevent stale data leaking between sessions
    queryClient.clear()
  }, [queryClient])

  // SEC-S4-10: Auto-logout on inactivity
  const warningToastId = useRef<string | number | undefined>(undefined)

  useEffect(() => {
    if (!isAuthenticated) return

    let logoutTimer: ReturnType<typeof setTimeout>
    let warningTimer: ReturnType<typeof setTimeout>

    const resetTimers = () => {
      clearTimeout(logoutTimer)
      clearTimeout(warningTimer)

      // Dismiss the warning toast if the user became active again
      if (warningToastId.current !== undefined) {
        toast.dismiss(warningToastId.current)
        warningToastId.current = undefined
      }

      warningTimer = setTimeout(() => {
        warningToastId.current = toast.warning(
          "You will be logged out in 5 minutes due to inactivity.",
          { duration: 5 * 60 * 1000, id: "inactivity-warning" }
        )
      }, INACTIVITY_WARNING_MS)

      logoutTimer = setTimeout(() => {
        toast.dismiss(warningToastId.current)
        warningToastId.current = undefined
        logout()
      }, INACTIVITY_TIMEOUT_MS)
    }

    // Start timers immediately
    resetTimers()

    // Reset on user activity
    for (const event of ACTIVITY_EVENTS) {
      window.addEventListener(event, resetTimers, { passive: true })
    }

    return () => {
      clearTimeout(logoutTimer)
      clearTimeout(warningTimer)
      for (const event of ACTIVITY_EVENTS) {
        window.removeEventListener(event, resetTimers)
      }
      if (warningToastId.current !== undefined) {
        toast.dismiss(warningToastId.current)
        warningToastId.current = undefined
      }
    }
  }, [isAuthenticated, logout])

  // Auto-logout when API detects session expiry (401 with failed refresh)
  useEffect(() => {
    const handleSessionExpired = () => {
      toast.error("Your session has expired. Please log in again.")
      logout()
    }

    window.addEventListener("auth:session-expired", handleSessionExpired)
    return () => {
      window.removeEventListener("auth:session-expired", handleSessionExpired)
    }
  }, [logout])

  const setCurrentOrg = useCallback((org: Organization) => {
    setCurrentOrgState(org)
    storeOrgId(org.id)
    // V101-10: Invalidate all React Query caches when switching orgs
    // so stale org-scoped data is refetched for the new org context
    queryClient.invalidateQueries()
  }, [queryClient])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated,
      isLoading,
      currentOrg,
      organizations,
      orgsLoading,
      orgsError,
      login,
      register,
      logout,
      setCurrentOrg,
      refreshUser,
      refreshOrgs,
    }),
    [
      user,
      isAuthenticated,
      isLoading,
      currentOrg,
      organizations,
      orgsLoading,
      orgsError,
      login,
      register,
      logout,
      setCurrentOrg,
      refreshUser,
      refreshOrgs,
    ]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
