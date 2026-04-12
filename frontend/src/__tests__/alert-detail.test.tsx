/**
 * Tests for alert detail view data flow.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { useUpdateAlert } from "@/features/alerts"
import { createAlert } from "@/test-fixtures"

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

describe("Alert detail data flow", () => {
  it("transitions alert from new to acknowledged", async () => {
    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ id: 1, status: "acknowledged" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(result.current.data?.data.status).toBe("acknowledged")
  })

  it("transitions alert from acknowledged to resolved", async () => {
    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ id: 1, status: "resolved" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(result.current.data?.data.status).toBe("resolved")
  })

  it("shows error when status update fails", async () => {
    server.use(
      http.patch("http://localhost:8080/api/alerts/:id", () => {
        return HttpResponse.json(
          { data: null, error: "Alert not found" },
          { status: 404 }
        )
      })
    )

    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({ id: 999, status: "acknowledged" })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })

  it("returns correct alert fields in response", async () => {
    server.use(
      http.patch("http://localhost:8080/api/alerts/:id", ({ params }) => {
        return HttpResponse.json({
          data: createAlert({
            id: Number(params.id),
            severity: "critical",
            status: "acknowledged",
            message: "Malicious code detected in install script",
            packageName: "evil-pkg",
            packageEcosystem: "npm",
          }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useUpdateAlert(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ id: 5, status: "acknowledged" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const alert = result.current.data?.data
    expect(alert?.id).toBe(5)
    expect(alert?.severity).toBe("critical")
    expect(alert?.message).toBe("Malicious code detected in install script")
    expect(alert?.packageName).toBe("evil-pkg")
  })
})
