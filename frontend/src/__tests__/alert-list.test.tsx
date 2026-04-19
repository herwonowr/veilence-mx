/**
 * Tests for alert list page component rendering.
 * Tests the AlertsContent component behavior with MSW-mocked API.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { useAlerts, useUpdateAlert } from "@/features/alerts"
import { createAlert, createAlerts } from "@/test-fixtures"

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

vi.mock("@/core/providers/auth-provider", () => ({
  useAuth: vi.fn(() => ({
    user: null,
    isAuthenticated: false,
    isLoading: false,
    currentWorkspace: null,
    workspaces: [],
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    setCurrentWorkspace: vi.fn(),
    refreshUser: vi.fn(),
    refreshWorkspaces: vi.fn(),
  })),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  // eslint-disable-next-line react/display-name
  return ({ children }: { children: React.ReactNode }) => {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe("Alert list data flow", () => {
  it("fetches alerts with severity filter", async () => {
    const criticalAlerts = [
      createAlert({ id: 1, severity: "critical", status: "new" }),
      createAlert({ id: 2, severity: "critical", status: "acknowledged" }),
    ]

    server.use(
      http.get("http://localhost:8080/api/alerts", ({ request }) => {
        const url = new URL(request.url)
        const severity = url.searchParams.get("severity")
        if (severity === "critical") {
          return HttpResponse.json({
            data: criticalAlerts,
            error: null,
            meta: { page: 1, limit: 20, total: 2 },
          })
        }
        return HttpResponse.json({
          data: createAlerts(5),
          error: null,
          meta: { page: 1, limit: 20, total: 5 },
        })
      })
    )

    const { result } = renderHook(
      () => useAlerts({ severity: "critical" }),
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(2)
    expect(result.current.data?.data.every((a) => a.severity === "critical")).toBe(true)
  })

  it("fetches alerts with status filter", async () => {
    const newAlerts = [createAlert({ id: 1, status: "new", severity: "high" })]

    server.use(
      http.get("http://localhost:8080/api/alerts", ({ request }) => {
        const url = new URL(request.url)
        if (url.searchParams.get("status") === "new") {
          return HttpResponse.json({
            data: newAlerts,
            error: null,
            meta: { page: 1, limit: 20, total: 1 },
          })
        }
        return HttpResponse.json({
          data: createAlerts(5),
          error: null,
          meta: { page: 1, limit: 20, total: 5 },
        })
      })
    )

    const { result } = renderHook(
      () => useAlerts({ status: "new" }),
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].status).toBe("new")
  })

  it("acknowledges an alert via mutation", async () => {
    server.use(
      http.patch("http://localhost:8080/api/alerts/:id", async ({ params, request }) => {
        const body = (await request.json()) as { status: string }
        return HttpResponse.json({
          data: createAlert({
            id: Number(params.id),
            status: body.status as "acknowledged",
          }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      const response = await result.current.mutateAsync({ id: 1, status: "acknowledged" })
      expect(response.data.status).toBe("acknowledged")
    })
  })

  it("resolves an alert via mutation", async () => {
    server.use(
      http.patch("http://localhost:8080/api/alerts/:id", async ({ params, request }) => {
        const body = (await request.json()) as { status: string }
        return HttpResponse.json({
          data: createAlert({
            id: Number(params.id),
            status: body.status as "resolved",
          }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      const response = await result.current.mutateAsync({ id: 1, status: "resolved" })
      expect(response.data.status).toBe("resolved")
    })
  })
})
