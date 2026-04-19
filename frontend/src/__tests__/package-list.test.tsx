/**
 * Tests for package list and creation data flow.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import { usePackages, useCreatePackage } from "@/features/packages"
import { createPackage, createPackages } from "@/test-fixtures"

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

describe("Package list data flow", () => {
  it("renders packages from API", async () => {
    const { result } = renderHook(() => usePackages(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const packages = result.current.data?.data
    expect(packages).toHaveLength(3)
    expect(packages?.[0]).toHaveProperty("name")
    expect(packages?.[0]).toHaveProperty("ecosystem")
    expect(packages?.[0]).toHaveProperty("latestVersion")
  })

  it("filters by ecosystem", async () => {
    server.use(
      http.get("http://localhost:8080/api/packages", ({ request }) => {
        const url = new URL(request.url)
        const ecosystem = url.searchParams.get("ecosystem")
        if (ecosystem === "npm") {
          return HttpResponse.json({
            data: [createPackage({ id: 1, name: "lodash", ecosystem: "npm" })],
            error: null,
            meta: { page: 1, limit: 20, total: 1 },
          })
        }
        return HttpResponse.json({
          data: createPackages(3),
          error: null,
          meta: { page: 1, limit: 20, total: 3 },
        })
      })
    )

    const { result } = renderHook(
      () => usePackages({ ecosystem: "npm" }),
      { wrapper: createWrapper() }
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].ecosystem).toBe("npm")
  })

  it("shows empty state", async () => {
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
})

describe("Package creation form data flow", () => {
  it("creates a package via API", async () => {
    let capturedBody: { name: string; ecosystem: string } | null = null

    server.use(
      http.post("http://localhost:8080/api/packages", async ({ request }) => {
        capturedBody = (await request.json()) as { name: string; ecosystem: string }
        return HttpResponse.json(
          {
            data: createPackage({ name: capturedBody.name, ecosystem: capturedBody.ecosystem as "npm" | "python" }),
            error: null,
          },
          { status: 201 }
        )
      })
    )

    const { result } = renderHook(() => useCreatePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ name: "requests", ecosystem: "python" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(capturedBody?.name).toBe("requests")
    expect(capturedBody?.ecosystem).toBe("python")
  })

  it("handles duplicate package error", async () => {
    server.use(
      http.post("http://localhost:8080/api/packages", () => {
        return HttpResponse.json(
          { data: null, error: "Package already exists" },
          { status: 409 }
        )
      })
    )

    const { result } = renderHook(() => useCreatePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "existing-pkg", ecosystem: "npm" })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})
