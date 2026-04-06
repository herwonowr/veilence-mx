/**
 * Tests for organization management hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "./msw-server"
import {
  useOrganizations,
  useOrganization,
  useCreateOrganization,
  useOrgMembers,
  useOrgRoles,
} from "@/features/admin/hooks/use-organizations"
import { createOrg } from "@/test-fixtures"

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

describe("useOrganizations", () => {
  it("fetches organizations list", async () => {
    const { result } = renderHook(() => useOrganizations(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].name).toBe("Test Org")
  })

  it("handles empty organizations", async () => {
    server.use(
      http.get("http://localhost:8080/api/orgs", () => {
        return HttpResponse.json({ data: [], error: null })
      })
    )

    const { result } = renderHook(() => useOrganizations(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })
})

describe("useOrganization", () => {
  it("fetches single organization", async () => {
    const { result } = renderHook(() => useOrganization(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data.id).toBe(1)
    expect(result.current.data?.data.name).toBe("Test Org")
  })

  it("does not fetch when id is 0", () => {
    const { result } = renderHook(() => useOrganization(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("useCreateOrganization", () => {
  it("creates an organization", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs", async ({ request }) => {
        const body = (await request.json()) as { name: string; slug: string }
        return HttpResponse.json({
          data: createOrg({ name: body.name, slug: body.slug }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useCreateOrganization(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        name: "New Org",
        slug: "new-org",
        description: "A new organization",
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles duplicate slug error", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs", () => {
        return HttpResponse.json(
          { data: null, error: "Slug already taken" },
          { status: 409 }
        )
      })
    )

    const { result } = renderHook(() => useCreateOrganization(), {
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

describe("useOrgMembers", () => {
  it("fetches org members", async () => {
    const { result } = renderHook(() => useOrgMembers(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].role.name).toBe("owner")
  })
})

describe("useOrgRoles", () => {
  it("fetches org roles", async () => {
    const { result } = renderHook(() => useOrgRoles(1), {
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
