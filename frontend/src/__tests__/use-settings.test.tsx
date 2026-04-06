/**
 * Tests for useSettings and useUpdateSettings hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "./msw-server"
import { useSettings, useUpdateSettings } from "@/features/settings"

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

vi.mock("@/lib/auth-context", () => ({
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

describe("useSettings", () => {
  it("fetches settings successfully", async () => {
    const { result } = renderHook(() => useSettings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data.pypi_poll_interval).toBe("5m")
    expect(result.current.data?.data.npm_poll_interval).toBe("5m")
    expect(result.current.data?.data.analyzer_type).toBe("api")
  })

  it("handles API error", async () => {
    server.use(
      http.get("http://localhost:8080/api/settings", () => {
        return HttpResponse.json(
          { data: null, error: "Forbidden" },
          { status: 403 }
        )
      })
    )

    const { result } = renderHook(() => useSettings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("useUpdateSettings", () => {
  it("updates settings successfully", async () => {
    const { result } = renderHook(() => useUpdateSettings(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ pypi_poll_interval: "10m" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles update error", async () => {
    server.use(
      http.put("http://localhost:8080/api/settings", () => {
        return HttpResponse.json(
          { data: null, error: "Failed to save" },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useUpdateSettings(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({ bad_setting: "value" })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})
