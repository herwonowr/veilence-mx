/**
 * Tests for notification channel management hooks.
 */

import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { http, HttpResponse } from "msw"
import { server } from "@/__tests__/msw-server"
import {
  useChannels,
  useCreateChannel,
  useDeleteChannel,
} from "@/features/notifications"
import { createNotificationChannel } from "@/test-fixtures"

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

describe("useChannels", () => {
  it("fetches notification channels for org", async () => {
    const { result } = renderHook(() => useChannels(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].name).toBe("Slack #alerts")
    expect(result.current.data?.data[0].type).toBe("slack")
  })

  it("does not fetch when orgId is null", () => {
    const { result } = renderHook(() => useChannels(null), {
      wrapper: createWrapper(),
    })

    expect(result.current.fetchStatus).toBe("idle")
  })

  it("handles empty channels list", async () => {
    server.use(
      http.get("http://localhost:8080/api/orgs/:orgId/notification-channels", () => {
        return HttpResponse.json({ data: [], error: null })
      })
    )

    const { result } = renderHook(() => useChannels(1), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.data).toEqual([])
  })
})

describe("useCreateChannel", () => {
  it("creates a slack channel", async () => {
    const { result } = renderHook(() => useCreateChannel(1), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        name: "Slack #security",
        type: "slack",
        config: JSON.stringify({ webhookUrl: "https://hooks.slack.com/test" }),
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("creates a webhook channel", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs/:orgId/notification-channels", async ({ request }) => {
        const body = (await request.json()) as { name: string; type: string; config: string }
        return HttpResponse.json(
          {
            data: createNotificationChannel({
              name: body.name,
              type: body.type as "webhook",
              config: body.config,
            }),
            error: null,
          },
          { status: 201 }
        )
      })
    )

    const { result } = renderHook(() => useCreateChannel(1), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        name: "Security Webhook",
        type: "webhook",
        config: JSON.stringify({ url: "https://example.com/webhook", secret: "test" }),
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("creates an email channel", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs/:orgId/notification-channels", async ({ request }) => {
        const body = (await request.json()) as { name: string; type: string; config: string }
        return HttpResponse.json(
          {
            data: createNotificationChannel({
              name: body.name,
              type: body.type as "email",
              config: body.config,
            }),
            error: null,
          },
          { status: 201 }
        )
      })
    )

    const { result } = renderHook(() => useCreateChannel(1), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        name: "Email Alerts",
        type: "email",
        config: JSON.stringify({ recipients: ["admin@example.com"] }),
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })

  it("handles creation error", async () => {
    server.use(
      http.post("http://localhost:8080/api/orgs/:orgId/notification-channels", () => {
        return HttpResponse.json(
          { data: null, error: "Invalid configuration" },
          { status: 400 }
        )
      })
    )

    const { result } = renderHook(() => useCreateChannel(1), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({
          name: "Bad Channel",
          type: "slack",
          config: "invalid-json",
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

describe("useDeleteChannel", () => {
  it("deletes a channel", async () => {
    const { result } = renderHook(() => useDeleteChannel(1), {
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
      http.delete("http://localhost:8080/api/orgs/:orgId/notification-channels/:id", () => {
        return HttpResponse.json(
          { data: null, error: "Not found" },
          { status: 404 }
        )
      })
    )

    const { result } = renderHook(() => useDeleteChannel(1), {
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
