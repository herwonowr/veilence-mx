/**
 * Tests for useRelease hook.
 *
 * Covers: successful fetch, disabled state, error handling.
 */

import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { useRelease, releaseKeys } from "@/features/packages"
import { createRelease } from "@/test-fixtures"

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

describe("useRelease", () => {
  it("fetches release detail successfully", async () => {
    const mockRelease = createRelease({ id: 42, version: "2.0.0" })
    server.use(
      http.get("http://localhost:8080/api/releases/42", () => {
        return HttpResponse.json({ data: mockRelease, error: null })
      })
    )

    const { result } = renderHook(() => useRelease(42), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual(mockRelease)
  })

  it("does not fetch when id is 0", async () => {
    const { result } = renderHook(() => useRelease(0), {
      wrapper: createWrapper(),
    })

    // Should never become loading or successful because enabled=false
    expect(result.current.fetchStatus).toBe("idle")
  })

  it("handles API error", async () => {
    server.use(
      http.get("http://localhost:8080/api/releases/99", () => {
        return HttpResponse.json(
          { data: null, error: "Release not found" },
          { status: 404 }
        )
      })
    )

    const { result } = renderHook(() => useRelease(99), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("releaseKeys", () => {
  it("generates correct key structure", () => {
    expect(releaseKeys.all).toEqual(["releases"])
    expect(releaseKeys.detail(5)).toEqual(["releases", "detail", 5])
  })
})
