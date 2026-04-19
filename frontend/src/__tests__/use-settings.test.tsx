/**
 * Tests for useSettings and useUpdateSettings hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { useSettings, useUpdateSettings } from "@/features/settings"

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
  return ({ children }: { children: React.ReactNode }) => {
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

    expect(result.current.data?.data.monitoring_interval).toBe("1h")
    expect(result.current.data?.data.discovery_scan_depth).toBe("50")
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
      result.current.mutate({ monitoring_interval: "30m" })
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
