"use client"

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react"
import { useQueryClient } from "@tanstack/react-query"
import type { User, Organization } from "@/types"
import {
  apiLogin,
  apiRegister,
  apiLogout,
  apiGetMe,
  apiGetOrgs,
  getStoredAccessToken,
  getStoredRefreshToken,
  storeTokens,
  clearTokens,
  getStoredOrgId,
  storeOrgId,
  clearOrgId,
} from "@/lib/api-client"

interface AuthContextValue {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  currentOrg: Organization | null
  organizations: Organization[]
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

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return ctx
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient()
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [currentOrg, setCurrentOrgState] = useState<Organization | null>(null)
  const [organizations, setOrganizations] = useState<Organization[]>([])

  const isAuthenticated = !!user

  const refreshUser = useCallback(async () => {
    try {
      const { data } = await apiGetMe()
      setUser(data)
    } catch {
      setUser(null)
      clearTokens()
    }
  }, [])

  const refreshOrgs = useCallback(async () => {
    try {
      const { data } = await apiGetOrgs()
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
    } catch {
      setOrganizations([])
    }
  }, [])

  // Initialize auth state from localStorage
  useEffect(() => {
    const token = getStoredAccessToken()
    if (token) {
      Promise.all([refreshUser(), refreshOrgs()]).finally(() => {
        setIsLoading(false)
      })
    } else {
      setIsLoading(false)
    }
  }, [refreshUser, refreshOrgs])

  // Auto-refresh token before expiry (refresh every 10 minutes)
  useEffect(() => {
    if (!isAuthenticated) return

    const interval = setInterval(
      async () => {
        const refreshToken = getStoredRefreshToken()
        if (!refreshToken) return

        try {
          const response = await fetch(
            `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/auth/refresh`,
            {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ refreshToken }),
            }
          )

          if (response.ok) {
            const body = await response.json()
            if (body.data?.accessToken && body.data?.refreshToken) {
              storeTokens(body.data.accessToken, body.data.refreshToken)
            }
          }
        } catch {
          // Token refresh failed silently
        }
      },
      10 * 60 * 1000
    )

    return () => clearInterval(interval)
  }, [isAuthenticated])

  const login = useCallback(
    async (email: string, password: string) => {
      const { data } = await apiLogin(email, password)
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
      const { data } = await apiRegister(params)
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
        await apiLogout(refreshToken)
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

  const setCurrentOrg = useCallback((org: Organization) => {
    setCurrentOrgState(org)
    storeOrgId(org.id)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated,
      isLoading,
      currentOrg,
      organizations,
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
