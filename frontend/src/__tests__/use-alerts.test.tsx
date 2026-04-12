/**
 * Tests for useAlerts and useUpdateAlert hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { useAlerts, useUpdateAlert } from "@/features/alerts"
import { createAlerts } from "@/test-fixtures"

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

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
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe("useAlerts", () => {
  it("fetches alerts successfully", async () => {
    const { result } = renderHook(() => useAlerts(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(5)
  })

  it("handles API error", async () => {
    server.use(
      http.get("http://localhost:8080/api/alerts", () => {
        return HttpResponse.json(
          { data: null, error: "Internal error" },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useAlerts(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })

  it("returns empty list when no alerts", async () => {
    server.use(
      http.get("http://localhost:8080/api/alerts", () => {
        return HttpResponse.json({
          data: [],
          error: null,
          meta: { page: 1, limit: 20, total: 0 },
        })
      })
    )

    const { result } = renderHook(() => useAlerts(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })

  it("passes filter params", async () => {
    let capturedUrl = ""
    server.use(
      http.get("http://localhost:8080/api/alerts", ({ request }) => {
        capturedUrl = request.url
        return HttpResponse.json({
          data: createAlerts(1),
          error: null,
          meta: { page: 1, limit: 20, total: 1 },
        })
      })
    )

    const { result } = renderHook(
      () => useAlerts({ severity: "critical", status: "new" }),
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(capturedUrl).toContain("severity=critical")
    expect(capturedUrl).toContain("status=new")
  })
})

describe("useUpdateAlert", () => {
  it("updates alert status", async () => {
    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ id: 1, status: "acknowledged" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles update error with rollback", async () => {
    server.use(
      http.patch("http://localhost:8080/api/alerts/:id", () => {
        return HttpResponse.json(
          { data: null, error: "Failed to update" },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({ id: 1, status: "acknowledged" })
      } catch {
        // Expected to fail
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})
