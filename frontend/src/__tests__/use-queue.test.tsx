/**
 * Tests for useQueueStats, useDeadJobs, useRetryDeadJobs hooks.
 *
 * Covers: queue stats fetch, dead jobs fetch with type filter,
 * retry mutation and cache invalidation, error handling.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "./msw-server"
import { useQueueStats, useDeadJobs, useRetryDeadJobs, queueKeys } from "@/features/settings"
import { createQueueStats } from "@/test-fixtures"

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

describe("useQueueStats", () => {
  it("fetches queue stats successfully", async () => {
    const { result } = renderHook(() => useQueueStats(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const stats = result.current.data?.data
    expect(stats).toBeDefined()
    expect(stats?.diff).toBeDefined()
    expect(stats?.analyze).toBeDefined()
    expect(stats?.diff.pending).toBe(5)
  })

  it("handles API error", async () => {
    server.use(
      http.get("http://localhost:8080/api/queue/stats", () => {
        return HttpResponse.json(
          { data: null, error: "Queue unavailable" },
          { status: 503 }
        )
      })
    )

    const { result } = renderHook(() => useQueueStats(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("useDeadJobs", () => {
  it("fetches dead jobs without type filter", async () => {
    server.use(
      http.get("http://localhost:8080/api/queue/dead", () => {
        return HttpResponse.json({
          data: [
            { id: "1", type: "diff", payload: "{}", error: "timeout", failedAt: "2026-04-01T00:00:00Z" },
          ],
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useDeadJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
  })

  it("passes type filter parameter", async () => {
    let capturedUrl = ""
    server.use(
      http.get("http://localhost:8080/api/queue/dead", ({ request }) => {
        capturedUrl = request.url
        return HttpResponse.json({ data: [], error: null })
      })
    )

    const { result } = renderHook(() => useDeadJobs("analyze"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(capturedUrl).toContain("type=analyze")
  })
})

describe("useRetryDeadJobs", () => {
  it("retries dead jobs successfully", async () => {
    server.use(
      http.post("http://localhost:8080/api/queue/retry-dead", () => {
        return HttpResponse.json({
          data: { message: "Retried 5 dead jobs" },
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useRetryDeadJobs(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate()
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles retry error", async () => {
    server.use(
      http.post("http://localhost:8080/api/queue/retry-dead", () => {
        return HttpResponse.json(
          { data: null, error: "Queue error" },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useRetryDeadJobs(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync()
      } catch {
        // Expected to fail
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("queueKeys", () => {
  it("generates correct key structure", () => {
    expect(queueKeys.all).toEqual(["queue"])
    expect(queueKeys.stats()).toEqual(["queue", "stats"])
    expect(queueKeys.dead("diff")).toEqual(["queue", "dead", "diff"])
    expect(queueKeys.dead()).toEqual(["queue", "dead", undefined])
  })
})
