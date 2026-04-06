/**
 * Tests for package creation form validation.
 */

import { renderHook, act, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "./msw-server"
import { useCreatePackage } from "@/features/packages"
import { createPackage } from "@/test-fixtures"

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

describe("Package creation form", () => {
  it("submits with name and registry", async () => {
    let capturedBody: Record<string, string> | null = null

    server.use(
      http.post("http://localhost:8080/api/packages", async ({ request }) => {
        capturedBody = (await request.json()) as Record<string, string>
        return HttpResponse.json(
          {
            data: createPackage({ name: capturedBody.name, registry: capturedBody.registry as "npm" | "pypi" }),
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
      result.current.mutate({ name: "lodash", registry: "npm" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
    expect(capturedBody).toEqual({ name: "lodash", registry: "npm" })
  })

  it("handles server validation error", async () => {
    server.use(
      http.post("http://localhost:8080/api/packages", () => {
        return HttpResponse.json(
          { data: null, error: "Invalid package name" },
          { status: 400 }
        )
      })
    )

    const { result } = renderHook(() => useCreatePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "", registry: "npm" })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })

  it("creates npm package", async () => {
    const { result } = renderHook(() => useCreatePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ name: "express", registry: "npm" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("creates pypi package", async () => {
    const { result } = renderHook(() => useCreatePackage(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ name: "requests", registry: "pypi" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})
