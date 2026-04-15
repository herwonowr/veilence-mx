/**
 * Tests for organization management hooks from features/admin.
 *
 * Covers: useOrganizations, useOrganization, useCreateOrganization,
 * useUpdateOrganization, useDeleteOrganization, useOrgMembers, useOrgRoles,
 * usePermissions, useInviteMember, useRemoveMember, useUpdateMemberRole,
 * useAuditLogs.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import {
  useOrganizations,
  useOrganization,
  useCreateOrganization,
  useUpdateOrganization,
  useDeleteOrganization,
  useOrgMembers,
  useOrgRoles,
  usePermissions,
  useInviteMember,
  useRemoveMember,
  useUpdateMemberRole,
  useAuditLogs,
  orgKeys,
} from "@/features/admin"
import { createOrg } from "@/test-fixtures"

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

describe("useOrganizations", () => {
  it("fetches organizations list", async () => {
    const { result } = renderHook(() => useOrganizations(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data?.[0]?.name).toBe("Test Org")
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

    expect(result.current.data?.data?.id).toBe(1)
  })

  it("does not fetch when id is 0", () => {
    const { result } = renderHook(() => useOrganization(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("useCreateOrganization", () => {
  it("creates organization successfully", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs", async ({ request }) => {
        const body = (await request.json()) as { name: string; slug: string }
        return HttpResponse.json(
          {
            data: createOrg({ name: body.name, slug: body.slug }),
            error: null,
          },
          { status: 201 }
        )
      })
    )

    const { result } = renderHook(() => useCreateOrganization(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ name: "New Org", slug: "new-org" })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles creation error", async () => {
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
        await result.current.mutateAsync({ name: "Dup", slug: "dup" })
      } catch {
        // Expected
      }
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
  })
})

describe("useUpdateOrganization", () => {
  it("updates organization", async () => {
    server.use(
      http.put("http://localhost:8080/api/orgs/:id", () => {
        return HttpResponse.json({
          data: createOrg({ name: "Updated" }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useUpdateOrganization(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ id: 1, data: { name: "Updated" } })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useDeleteOrganization", () => {
  it("deletes organization", async () => {
    server.use(
      http.delete("http://localhost:8080/api/orgs/:id", () => {
        return HttpResponse.json({ data: null, error: null })
      })
    )

    const { result } = renderHook(() => useDeleteOrganization(), {
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

describe("useOrgMembers", () => {
  it("fetches org members", async () => {
    const { result } = renderHook(() => useOrgMembers(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
  })

  it("does not fetch when orgId is 0", () => {
    const { result } = renderHook(() => useOrgMembers(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
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

    expect(result.current.data?.data).toHaveLength(4)
  })
})

describe("usePermissions", () => {
  it("fetches permissions list", async () => {
    server.use(
      http.get("http://localhost:8080/api/permissions", () => {
        return HttpResponse.json({
          data: [
            { id: 1, resource: "packages", action: "read" },
            { id: 2, resource: "packages", action: "write" },
          ],
          error: null,
        })
      })
    )

    const { result } = renderHook(() => usePermissions(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(2)
  })
})

describe("useInviteMember", () => {
  it("sends invitation", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs/:orgId/invitations", () => {
        return HttpResponse.json(
          { data: { token: "invite-token-123" }, error: null },
          { status: 201 }
        )
      })
    )

    const { result } = renderHook(() => useInviteMember(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        orgId: 1,
        data: { email: "newmember@example.com", roleId: 3 },
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useRemoveMember", () => {
  it("removes member", async () => {
    server.use(
      http.delete("http://localhost:8080/api/orgs/:orgId/members/:userId", () => {
        return HttpResponse.json({ data: null, error: null })
      })
    )

    const { result } = renderHook(() => useRemoveMember(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ orgId: 1, userId: 2 })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useUpdateMemberRole", () => {
  it("updates member role", async () => {
    server.use(
      http.put("http://localhost:8080/api/orgs/:orgId/members/:userId/role", () => {
        return HttpResponse.json({ data: null, error: null })
      })
    )

    const { result } = renderHook(() => useUpdateMemberRole(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ orgId: 1, userId: 2, roleId: 4 })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useAuditLogs", () => {
  it("fetches audit logs for org", async () => {
    server.use(
      http.get("http://localhost:8080/api/orgs/:orgId/audit-logs", () => {
        return HttpResponse.json({
          data: [
            {
              id: 1,
              userId: 1,
              orgId: 1,
              action: "create",
              resource: "package",
              resourceId: 1,
              details: "created package",
              createdAt: "2026-04-01T00:00:00Z",
            },
          ],
          error: null,
          meta: { page: 1, limit: 20, total: 1 },
        })
      })
    )

    const { result } = renderHook(() => useAuditLogs(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
  })

  it("does not fetch when orgId is 0", () => {
    const { result } = renderHook(() => useAuditLogs(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("orgKeys", () => {
  it("generates correct key structure", () => {
    expect(orgKeys.all).toEqual(["organizations"])
    expect(orgKeys.lists()).toEqual(["organizations", "list"])
    expect(orgKeys.detail(1)).toEqual(["organizations", "detail", 1])
    expect(orgKeys.members(1)).toEqual(["organizations", "members", 1])
    expect(orgKeys.roles(1)).toEqual(["organizations", "roles", 1])
    expect(orgKeys.permissions()).toEqual(["organizations", "permissions"])
  })
})
