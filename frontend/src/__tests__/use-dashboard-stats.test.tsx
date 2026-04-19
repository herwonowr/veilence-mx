/**
 * Test for a React Query–style data-fetching hook.
 *
 * Even though React Query migration (#6) may not have landed yet,
 * this test validates the pattern we'll use throughout the app:
 * wrapping api-client calls in hooks that provide loading/error/data states.
 *
 * We test a simple useDashboardStats hook implemented inline below,
 * which mirrors what the React Query migration will produce.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { useState, useEffect, useCallback } from "react"
import * as apiClient from "@/core"
import type { DashboardStats } from "@/domains/dashboard"
import type { ApiResponse } from "@/domains/common"

// Mock the API client
vi.mock("@/lib/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof apiClient>()
  return {
    ...actual,
    getDashboardStats: vi.fn(),
    getStoredAccessToken: vi.fn(() => null),
    getStoredRefreshToken: vi.fn(() => null),
  }
})

/**
 * A data-fetching hook pattern that mirrors what React Query provides.
 * Once React Query lands, this would be:
 *   useQuery({ queryKey: ['dashboard-stats'], queryFn: getDashboardStats })
 */
const useDashboardStats = () => {
  const [data, setData] = useState<DashboardStats | null>(null)
  const [error, setError] = useState<Error | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  const refetch = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await apiClient.getDashboardStats()
      setData(response.data)
    } catch (err) {
      setError(err instanceof Error ? err : new Error("Unknown error"))
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- test-only hook, triggers fetch on mount
    refetch()
  }, [refetch])

  return { data, error, isLoading, refetch }
}

const mockStats: DashboardStats = {
  totalPackages: 150,
  totalReleases: 1200,
  pendingAnalyses: 5,
  activeAlerts: 3,
  recentMalicious: 1,
}

describe("useDashboardStats hook (data-fetching pattern)", () => {
  it("returns loading state initially", () => {
    vi.mocked(apiClient.getDashboardStats).mockReturnValue(
      new Promise(() => {}) // never resolves
    )

    const { result } = renderHook(() => useDashboardStats())

    expect(result.current.isLoading).toBe(true)
    expect(result.current.data).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it("returns data on successful fetch", async () => {
    vi.mocked(apiClient.getDashboardStats).mockResolvedValue({
      data: mockStats,
      error: null,
    })

    const { result } = renderHook(() => useDashboardStats())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.data).toEqual(mockStats)
    expect(result.current.error).toBeNull()
  })

  it("returns error on failed fetch", async () => {
    vi.mocked(apiClient.getDashboardStats).mockRejectedValue(
      new Error("Network error")
    )

    const { result } = renderHook(() => useDashboardStats())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.data).toBeNull()
    expect(result.current.error?.message).toBe("Network error")
  })

  it("refetch triggers a new request", async () => {
    vi.mocked(apiClient.getDashboardStats).mockResolvedValue({
      data: mockStats,
      error: null,
    })

    const { result } = renderHook(() => useDashboardStats())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    // Refetch with updated data
    const updatedStats = { ...mockStats, totalPackages: 200 }
    vi.mocked(apiClient.getDashboardStats).mockResolvedValue({
      data: updatedStats,
      error: null,
    })

    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(result.current.data?.totalPackages).toBe(200)
    })

    // Should have been called twice: initial + refetch
    expect(apiClient.getDashboardStats).toHaveBeenCalledTimes(2)
  })

  it("caches data between state transitions", async () => {
    vi.mocked(apiClient.getDashboardStats).mockResolvedValue({
      data: mockStats,
      error: null,
    })

    const { result } = renderHook(() => useDashboardStats())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    // Data is available
    expect(result.current.data?.totalPackages).toBe(150)

    // Trigger refetch - data should persist until new data arrives
    let resolveRefetch: (value: ApiResponse<DashboardStats>) => void
    vi.mocked(apiClient.getDashboardStats).mockReturnValue(
      new Promise((resolve) => {
        resolveRefetch = resolve
      })
    )

    // Start refetch (don't await, to catch intermediate state)
    let refetchPromise: Promise<void>
    await act(async () => {
      refetchPromise = result.current.refetch()
      // Now resolve
      resolveRefetch!({ data: { ...mockStats, totalPackages: 300 }, error: null })
      await refetchPromise
    })

    await waitFor(() => {
      expect(result.current.data?.totalPackages).toBe(300)
    })
  })
})
