/**
 * Tests for settings form data flow.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { useSettings, useUpdateSettings, useReanalyzeAll } from "@/features/settings"

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

describe("Settings form data flow", () => {
  it("loads current settings", async () => {
    const { result } = renderHook(() => useSettings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual({
      python_poll_interval: "5m",
      npm_poll_interval: "5m",
      max_concurrent_analyses: "3",
      analyzer_type: "api",
    })
  })

  it("updates settings with new values", async () => {
    let capturedBody: Record<string, string> | null = null

    server.use(
      http.put("http://localhost:8080/api/settings", async ({ request }) => {
        capturedBody = (await request.json()) as Record<string, string>
        return HttpResponse.json({ data: capturedBody, error: null })
      })
    )

    const { result } = renderHook(() => useUpdateSettings(), {
      wrapper: createWrapper(),
    })

    const newSettings = {
      python_poll_interval: "10m",
      npm_poll_interval: "15m",
      max_concurrent_analyses: "5",
      analyzer_type: "copilot",
    }

    await act(async () => {
      result.current.mutate(newSettings)
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(capturedBody).toEqual(newSettings)
  })

  it("handles save error", async () => {
    server.use(
      http.put("http://localhost:8080/api/settings", () => {
        return HttpResponse.json(
          { data: null, error: "Permission denied" },
          { status: 403 }
        )
      })
    )

    const { result } = renderHook(() => useUpdateSettings(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({ python_poll_interval: "invalid" })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("Re-analyze action", () => {
  it("triggers re-analysis", async () => {
    const { result } = renderHook(() => useReanalyzeAll(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate()
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(result.current.data?.data.queued).toBe(10)
  })

  it("handles re-analysis error", async () => {
    server.use(
      http.post("http://localhost:8080/api/sync/reanalyze", () => {
        return HttpResponse.json(
          { data: null, error: "Queue unavailable" },
          { status: 503 }
        )
      })
    )

    const { result } = renderHook(() => useReanalyzeAll(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync()
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})
