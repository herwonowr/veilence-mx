/**
 * Tests for usePackages, useCreatePackage, useDeletePackage hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { usePackages, useCreatePackage, useDeletePackage } from "@/features/packages"
import { createPackages } from "@/test-fixtures"

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

describe("usePackages", () => {
  it("fetches packages successfully", async () => {
    const { result } = renderHook(() => usePackages(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(3)
    expect(result.current.data?.data[0].name).toBe("package-1")
  })

  it("handles API error", async () => {
    server.use(
      http.get("http://localhost:8080/api/packages", () => {
        return HttpResponse.json(
          { data: null, error: "Server error" },
          { status: 500 }
        )
      })
    )

    const { result } = renderHook(() => usePackages(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })

  it("returns empty array when no packages", async () => {
    server.use(
      http.get("http://localhost:8080/api/packages", () => {
        return HttpResponse.json({
          data: [],
          error: null,
          meta: { page: 1, limit: 20, total: 0 },
        })
      })
    )

    const { result } = renderHook(() => usePackages(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
    expect(result.current.data?.meta?.total).toBe(0)
  })

  it("passes query params through", async () => {
    let capturedUrl = ""
    server.use(
      http.get("http://localhost:8080/api/packages", ({ request }) => {
        capturedUrl = request.url
        return HttpResponse.json({
          data: createPackages(1),
          error: null,
          meta: { page: 1, limit: 10, total: 1 },
        })
      })
    )

    const { result } = renderHook(
      () => usePackages({ ecosystem: "npm", search: "lodash", page: 2, limit: 10 }),
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(capturedUrl).toContain("ecosystem=npm")
    expect(capturedUrl).toContain("search=lodash")
    expect(capturedUrl).toContain("page=2")
    expect(capturedUrl).toContain("limit=10")
  })
})

describe("useCreatePackage", () => {
  it("creates a package successfully", async () => {
    const { result } = renderHook(() => useCreatePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ name: "new-pkg", ecosystem: "npm" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useDeletePackage", () => {
  it("deletes a package successfully", async () => {
    const { result } = renderHook(() => useDeletePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate(1)
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})
