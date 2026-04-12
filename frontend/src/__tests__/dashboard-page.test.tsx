/**
 * Tests for the Dashboard page component.
 */

import { waitFor } from "@/test-utils"
import { http, HttpResponse } from "msw"
import { server } from "./msw-server"

// Mock auth context to provide authenticated state
vi.mock("@/lib/auth-context", () => ({
  useAuth: vi.fn(),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/",
  useSearchParams: () => new URLSearchParams(),
}))

// Instead of testing the full page (which requires ProtectedRoute),
// we test the dashboard hooks with MSW and verify data flow
import { useDashboardStats, useChartData, useRecentReleases } from "@/features/dashboard"
import { renderHook } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe("Dashboard data hooks", () => {
  it("useDashboardStats returns stats from API", async () => {
    const { result } = renderHook(() => useDashboardStats(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const stats = result.current.data?.data
    expect(stats?.totalPackages).toBe(150)
    expect(stats?.totalReleases).toBe(1200)
    expect(stats?.pendingAnalyses).toBe(5)
    expect(stats?.activeAlerts).toBe(3)
    expect(stats?.recentMalicious).toBe(1)
  })

  it("useDashboardStats handles server error", async () => {
    server.use(
      http.get("http://localhost:8080/api/dashboard/stats", () => {
        return HttpResponse.json(
          { data: null, error: "Internal error" },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => useDashboardStats(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })

  it("useChartData returns chart data from API", async () => {
    const { result } = renderHook(() => useChartData(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const data = result.current.data?.data
    expect(data?.releaseActivity).toBeDefined()
    expect(data?.classifications).toBeDefined()
    expect(data?.ecosystems).toBeDefined()
    expect(data?.alertsBySeverity).toBeDefined()
  })

  it("useRecentReleases returns releases from API", async () => {
    const { result } = renderHook(
      () => useRecentReleases({ page: 1, limit: 15, latestPerPackage: true }),
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const releases = result.current.data?.data
    expect(releases).toHaveLength(3)
    expect(releases?.[0].packageName).toBe("requests")
    expect(releases?.[1].packageName).toBe("lodash")
    expect(releases?.[2].packageName).toBe("flask")
  })

  it("useRecentReleases handles empty releases", async () => {
    server.use(
      http.get("http://localhost:8080/api/dashboard/recent-releases", () => {
        return HttpResponse.json({
          data: [],
          error: null,
          meta: { page: 1, limit: 20, total: 0 },
        })
      })
    )

    const { result } = renderHook(() => useRecentReleases(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })
})
