/**
 * Tests for workspace management hooks from features/admin.
 *
 * Covers: useWorkspaces, useWorkspace, useCreateWorkspace,
 * useUpdateWorkspace, useDeleteWorkspace, useWorkspaceMembers, useWorkspaceRoles,
 * usePermissions, useInviteMember, useRemoveMember, useUpdateMemberRole,
 * useAuditLogs.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import {
  useWorkspaces,
  useWorkspace,
  useCreateWorkspace,
  useUpdateWorkspace,
  useDeleteWorkspace,
  useWorkspaceMembers,
  useWorkspaceRoles,
  usePermissions,
  useInviteMember,
  useRemoveMember,
  useUpdateMemberRole,
  useAuditLogs,
  workspaceKeys,
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
    expect(result.current.data?.data?.[0]?.name).toBe("Test Org")
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

    expect(result.current.data?.data?.id).toBe(1)
  })

  it("does not fetch when id is 0", () => {
    const { result } = renderHook(() => useWorkspace(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("useCreateWorkspace", () => {
  it("creates workspace successfully", async () => {
    server.use(
      http.post("http://localhost:8080/api/workspaces", async ({ request }) => {
        const body = (await request.json()) as { name: string; slug: string }
        return HttpResponse.json(
          {
            data: createWorkspace({ name: body.name, slug: body.slug }),
            error: null,
          },
          { status: 201 }
        )
      })
    )

    const { result } = renderHook(() => useCreateWorkspace(), {
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

describe("useUpdateWorkspace", () => {
  it("updates workspace", async () => {
    server.use(
      http.put("http://localhost:8080/api/workspaces/:id", () => {
        return HttpResponse.json({
          data: createWorkspace({ name: "Updated" }),
          error: null,
        })
      })
    )

    const { result } = renderHook(() => useUpdateWorkspace(), {
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

describe("useDeleteWorkspace", () => {
  it("deletes workspace", async () => {
    server.use(
      http.delete("http://localhost:8080/api/workspaces/:id", () => {
        return HttpResponse.json({ data: null, error: null })
      })
    )

    const { result } = renderHook(() => useDeleteWorkspace(), {
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

describe("useWorkspaceMembers", () => {
  it("fetches org members", async () => {
    const { result } = renderHook(() => useWorkspaceMembers(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
  })

  it("does not fetch when workspaceId is 0", () => {
    const { result } = renderHook(() => useWorkspaceMembers(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
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
      http.post("http://localhost:8080/api/workspaces/:workspaceId/invitations", () => {
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
        workspaceId: 1,
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
      http.delete("http://localhost:8080/api/workspaces/:workspaceId/members/:userId", () => {
        return HttpResponse.json({ data: null, error: null })
      })
    )

    const { result } = renderHook(() => useRemoveMember(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ workspaceId: 1, userId: 2 })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useUpdateMemberRole", () => {
  it("updates member role", async () => {
    server.use(
      http.put("http://localhost:8080/api/workspaces/:workspaceId/members/:userId/role", () => {
        return HttpResponse.json({ data: null, error: null })
      })
    )

    const { result } = renderHook(() => useUpdateMemberRole(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({ workspaceId: 1, userId: 2, roleId: 4 })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})

describe("useAuditLogs", () => {
  it("fetches audit logs for org", async () => {
    server.use(
      http.get("http://localhost:8080/api/workspaces/:workspaceId/audit-logs", () => {
        return HttpResponse.json({
          data: [
            {
              id: 1,
              userId: 1,
              workspaceId: 1,
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

  it("does not fetch when workspaceId is 0", () => {
    const { result } = renderHook(() => useAuditLogs(0), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })
})

describe("workspaceKeys", () => {
  it("generates correct key structure", () => {
    expect(workspaceKeys.all).toEqual(["workspaces"])
    expect(workspaceKeys.lists()).toEqual(["workspaces", "list"])
    expect(workspaceKeys.detail(1)).toEqual(["workspaces", "detail", 1])
    expect(workspaceKeys.members(1)).toEqual(["workspaces", "members", 1])
    expect(workspaceKeys.roles(1)).toEqual(["workspaces", "roles", 1])
    expect(workspaceKeys.permissions()).toEqual(["workspaces", "permissions"])
  })
})
