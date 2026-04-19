/**
 * Tests for workspace management hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import {
  useWorkspaces,
  useWorkspace,
  useCreateWorkspace,
  useWorkspaceMembers,
  useWorkspaceRoles,
} from "@/features/admin"
import { createWorkspace } from "@/test-fixtures"

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

describe("useWorkspaces", () => {
  it("fetches workspaces list", async () => {
    const { result } = renderHook(() => useWorkspaces(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].name).toBe("Test Org")
  })

  it("handles empty workspaces", async () => {
    server.use(
      http.get("http://localhost:8080/api/workspaces", () => {
        return HttpResponse.json({ data: [], error: null })
      })
    )

    const { result } = renderHook(() => useWorkspaces(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })
})

describe("useWorkspace", () => {
  it("fetches single workspace", async () => {
    const { result } = renderHook(() => useWorkspace(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data.id).toBe(1)
    expect(result.current.data?.data.name).toBe("Test Org")
  })

  it("does not fetch when id is 0", () => {
    const { result } = renderHook(() => useWorkspace(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("useCreateWorkspace", () => {
  it("creates a workspace", async () => {
    server.use(
      http.post("http://localhost:8080/api/workspaces", async ({ request }) => {
        const body = (await request.json()) as { name: string; slug: string }
        return HttpResponse.json({
          data: createWorkspace({ name: body.name, slug: body.slug }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useCreateWorkspace(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        name: "New Org",
        slug: "new-org",
        description: "A new workspace",
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles duplicate slug error", async () => {
    server.use(
      http.post("http://localhost:8080/api/workspaces", () => {
        return HttpResponse.json(
          { data: null, error: "Slug already taken" },
          { status: 409 }
        )
      })
    )

    const { result } = renderHook(() => useCreateWorkspace(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({
          name: "Existing Org",
          slug: "existing-org",
        })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("useWorkspaceMembers", () => {
  it("fetches org members", async () => {
    const { result } = renderHook(() => useWorkspaceMembers(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].role.name).toBe("owner")
  })
})

describe("useWorkspaceRoles", () => {
  it("fetches org roles", async () => {
    const { result } = renderHook(() => useWorkspaceRoles(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    const roles = result.current.data?.data
    expect(roles).toHaveLength(4)
    const roleNames = roles?.map((r) => r.name)
    expect(roleNames).toContain("owner")
    expect(roleNames).toContain("admin")
    expect(roleNames).toContain("member")
    expect(roleNames).toContain("viewer")
  })
})
