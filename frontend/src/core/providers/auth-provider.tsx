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
// Core-local interfaces - auth-provider needs User/Workspace shapes but core/ cannot import domains/
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

interface Workspace {
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
  getStoredWorkspaceId,
  storeWorkspaceId,
  clearWorkspaceId,
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
  currentWorkspace: Workspace | null
  workspaces: Workspace[]
  workspacesLoading: boolean
  workspacesError: string | null
  login: (email: string, password: string) => Promise<void>
  register: (data: {
    email: string
    password: string
    firstName: string
    lastName: string
  }) => Promise<void>
  logout: () => Promise<void>
  setCurrentWorkspace: (workspace: Workspace) => void
  refreshUser: () => Promise<void>
  refreshWorkspaces: () => Promise<void>
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
  const [currentWorkspace, setCurrentWorkspaceState] = useState<Workspace | null>(null)
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [workspacesLoading, setWorkspacesLoading] = useState(false)
  const [workspacesError, setWorkspacesError] = useState<string | null>(null)
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

  const refreshWorkspaces = useCallback(async () => {
    setWorkspacesLoading(true)
    setWorkspacesError(null)
    try {
      const { data } = await fetchApi<Workspace[]>("/api/workspaces")
      setWorkspaces(data ?? [])

      // Try to restore current workspace from localStorage
      const storedWorkspaceId = getStoredWorkspaceId()
      if (storedWorkspaceId && data) {
        const found = data.find((o) => o.id === storedWorkspaceId)
        if (found) {
          setCurrentWorkspaceState(found)
        } else if (data.length > 0) {
          setCurrentWorkspaceState(data[0])
          storeWorkspaceId(data[0].id)
        }
      } else if (data && data.length > 0) {
        setCurrentWorkspaceState(data[0])
        storeWorkspaceId(data[0].id)
      }
    } catch (err) {
      setWorkspaces([])
      const message = sanitizeErrorMessage(err, "Failed to load workspaces")
      setWorkspacesError(message)
      toast.error(message)
    } finally {
      setWorkspacesLoading(false)
    }
  }, [])

  // Initialize auth state from localStorage.
  // Extracted as a stable callback so the effect body contains no direct setState calls.
  const initAuth = useCallback(async () => {
    const token = getStoredAccessToken()
    if (token) {
      try {
        await Promise.all([refreshUser(), refreshWorkspaces()])
      } finally {
        setIsLoading(false)
      }
    } else {
      setIsLoading(false)
    }
  }, [refreshUser, refreshWorkspaces])

  const didInit = useRef(false)
  useEffect(() => {
    if (didInit.current) return
    didInit.current = true
    initAuth()
  }, [initAuth])

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
      await refreshWorkspaces()
    },
    [refreshWorkspaces]
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
      await refreshWorkspaces()
    },
    [refreshWorkspaces]
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
    clearWorkspaceId()
    setUser(null)
    setCurrentWorkspaceState(null)
    setWorkspaces([])
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

  const setCurrentWorkspace = useCallback((workspace: Workspace) => {
    setCurrentWorkspaceState(workspace)
    storeWorkspaceId(workspace.id)
    // V101-10: Invalidate all React Query caches when switching workspaces
    // so stale workspace-scoped data is refetched for the new workspace context
    queryClient.invalidateQueries()
  }, [queryClient])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated,
      isLoading,
      currentWorkspace,
      workspaces,
      workspacesLoading,
      workspacesError,
      login,
      register,
      logout,
      setCurrentWorkspace,
      refreshUser,
      refreshWorkspaces,
    }),
    [
      user,
      isAuthenticated,
      isLoading,
      currentWorkspace,
      workspaces,
      workspacesLoading,
      workspacesError,
      login,
      register,
      logout,
      setCurrentWorkspace,
      refreshUser,
      refreshWorkspaces,
    ]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
