/**
 * Tests for useNotifications, useUnreadCount, useMarkNotificationRead hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import {
  useNotifications,
  useUnreadCount,
  useMarkNotificationRead,
} from "@/features/notifications"

vi.mock("@/core/providers/auth-provider", () => ({
  useAuth: vi.fn(() => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    currentOrg: null,
    organizations: [],
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    setCurrentOrg: vi.fn(),
    refreshUser: vi.fn(),
    refreshOrgs: vi.fn(),
  })),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchInterval: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe("useUnreadCount", () => {
  it("fetches unread count", async () => {
    const { result } = renderHook(() => useUnreadCount(true), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data.count).toBe(3)
  })

  it("does not fetch when disabled", () => {
    const { result } = renderHook(() => useUnreadCount(false), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("useNotifications", () => {
  it("fetches notifications", async () => {
    const { result } = renderHook(() => useNotifications({ limit: 20 }), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data.length).toBeGreaterThan(0)
    expect(result.current.data?.data[0]).toHaveProperty("title")
    expect(result.current.data?.data[0]).toHaveProperty("message")
    expect(result.current.data?.data[0]).toHaveProperty("isRead")
  })

  it("handles empty notifications", async () => {
    server.use(
      http.get("http://localhost:8080/api/notifications", () => {
        return HttpResponse.json({
          data: [],
          error: null,
          meta: { page: 1, limit: 20, total: 0 },
        })
      })
    )

    const { result } = renderHook(() => useNotifications(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })
})

describe("useMarkNotificationRead", () => {
  it("marks notification as read", async () => {
    const { result } = renderHook(() => useMarkNotificationRead(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate(1)
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})
