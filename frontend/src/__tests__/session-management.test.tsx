/**
 * Tests for session management and API key hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "./msw-server"
import { useApiKeys, useCreateApiKey, useDeleteApiKey } from "@/features/account/hooks/use-api-keys"
import { useSessions } from "@/features/account/hooks/use-sessions"

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

describe("useApiKeys", () => {
  it("fetches API keys", async () => {
    const { result } = renderHook(() => useApiKeys(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(2)
    expect(result.current.data?.data[0].name).toBe("Production Key")
    expect(result.current.data?.data[1].name).toBe("CI Key")
  })

  it("handles empty API keys", async () => {
    server.use(
      http.get("http://localhost:8080/api/auth/api-keys", () => {
        return HttpResponse.json({ data: [], error: null })
      })
    )

    const { result } = renderHook(() => useApiKeys(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })
})

describe("useCreateApiKey", () => {
  it("creates an API key", async () => {
    const { result } = renderHook(() => useCreateApiKey(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ name: "New Key", scope: "read" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useDeleteApiKey", () => {
  it("deletes an API key", async () => {
    const { result } = renderHook(() => useDeleteApiKey(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate(1)
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles delete error", async () => {
    server.use(
      http.delete("http://localhost:8080/api/auth/api-keys/:id", () => {
        return HttpResponse.json(
          { data: null, error: "Not found" },
          { status: 404 }
        )
      })
    )

    const { result } = renderHook(() => useDeleteApiKey(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync(999)
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("useSessions", () => {
  it("fetches sessions", async () => {
    const { result } = renderHook(() => useSessions(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(2)
    expect(result.current.data?.data[0].ipAddress).toBe("127.0.0.1")
    expect(result.current.data?.data[1].ipAddress).toBe("192.168.1.100")
  })
})
