/**
 * MSW request handlers for Veilence-MX API endpoints.
 *
 * Default handlers return successful responses with realistic mock data.
 * Override per-test using: server.use(http.get(...))
 */

import { http, HttpResponse } from "msw"
import {
  createDashboardStats,
  createChartData,
  createRecentRelease,
  createPackages,
  createPackage,
  createAlerts,
  createAlert,
  createNotifications,
  createNotificationChannel,
  createApiKey,
  createSession,
  createQueueStats,
  createUser,
  createOrg,
} from "@/test-fixtures"

const API_BASE = "http://localhost:8080"

export const handlers = [
  // ── Dashboard ─────────────────────────────────────────────
  http.get(`${API_BASE}/api/dashboard/stats`, () => {
    return HttpResponse.json({
      data: createDashboardStats(),
      error: null,
    })
  }),

  http.get(`${API_BASE}/api/dashboard/charts`, () => {
    return HttpResponse.json({
      data: createChartData(),
      error: null,
    })
  }),

  http.get(`${API_BASE}/api/dashboard/recent-releases`, () => {
    return HttpResponse.json({
      data: [
        createRecentRelease({ id: 1, packageName: "requests", version: "2.32.0" }),
        createRecentRelease({ id: 2, packageName: "lodash", version: "4.17.22", packageEcosystem: "npm" }),
        createRecentRelease({ id: 3, packageName: "flask", version: "3.1.0", packageEcosystem: "python" }),
      ],
      error: null,
      meta: { page: 1, limit: 20, total: 3 },
    })
  }),

  // ── Packages ──────────────────────────────────────────────
  http.get(`${API_BASE}/api/packages`, () => {
    return HttpResponse.json({
      data: createPackages(3),
      error: null,
      meta: { page: 1, limit: 20, total: 3 },
    })
  }),

  http.get(`${API_BASE}/api/packages/:id`, ({ params }) => {
    return HttpResponse.json({
      data: createPackage({ id: Number(params.id) }),
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/packages`, async ({ request }) => {
    const body = (await request.json()) as { name: string; ecosystem: string }
    return HttpResponse.json(
      {
        data: createPackage({ name: body.name, ecosystem: body.ecosystem as "npm" | "python" }),
        error: null,
      },
      { status: 201 }
    )
  }),

  http.delete(`${API_BASE}/api/packages/:id`, () => {
    return HttpResponse.json({ data: null, error: null })
  }),

  http.get(`${API_BASE}/api/packages/:id/releases`, () => {
    return HttpResponse.json({
      data: [],
      error: null,
      meta: { page: 1, limit: 50, total: 0 },
    })
  }),

  // ── Alerts ────────────────────────────────────────────────
  http.get(`${API_BASE}/api/alerts`, () => {
    return HttpResponse.json({
      data: createAlerts(5),
      error: null,
      meta: { page: 1, limit: 20, total: 5 },
    })
  }),

  http.patch(`${API_BASE}/api/alerts/:id`, async ({ params, request }) => {
    const body = (await request.json()) as { status: string }
    return HttpResponse.json({
      data: createAlert({ id: Number(params.id), status: body.status as "new" | "acknowledged" | "resolved" }),
      error: null,
    })
  }),

  // ── Settings ──────────────────────────────────────────────
  http.get(`${API_BASE}/api/settings`, () => {
    return HttpResponse.json({
      data: {
        python_poll_interval: "5m",
        npm_poll_interval: "5m",
        max_concurrent_analyses: "3",
        analyzer_type: "api",
      },
      error: null,
    })
  }),

  http.put(`${API_BASE}/api/settings`, async ({ request }) => {
    const body = await request.json()
    return HttpResponse.json({ data: body, error: null })
  }),

  // ── Auth ──────────────────────────────────────────────────
  http.get(`${API_BASE}/api/auth/me`, () => {
    return HttpResponse.json({
      data: createUser(),
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/auth/login`, () => {
    return HttpResponse.json({
      data: {
        user: createUser(),
        accessToken: "mock-access-token",
        refreshToken: "mock-refresh-token",
      },
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/auth/register`, () => {
    return HttpResponse.json({
      data: {
        user: createUser(),
        accessToken: "mock-access-token",
        refreshToken: "mock-refresh-token",
      },
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/auth/forgot-password`, () => {
    return HttpResponse.json({
      data: { message: "If an account exists, a reset link has been sent." },
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/auth/reset-password`, () => {
    return HttpResponse.json({
      data: { message: "Password reset successfully." },
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/auth/refresh`, () => {
    return HttpResponse.json({
      data: {
        user: createUser(),
        accessToken: "refreshed-access-token",
        refreshToken: "refreshed-refresh-token",
      },
      error: null,
    })
  }),

  // ── Organizations ─────────────────────────────────────────
  http.get(`${API_BASE}/api/orgs`, () => {
    return HttpResponse.json({
      data: [createOrg()],
      error: null,
    })
  }),

  http.get(`${API_BASE}/api/orgs/:id`, ({ params }) => {
    return HttpResponse.json({
      data: createOrg({ id: Number(params.id) }),
      error: null,
    })
  }),

  http.get(`${API_BASE}/api/orgs/:id/members`, () => {
    return HttpResponse.json({
      data: [
        {
          id: 1,
          orgId: 1,
          userId: 1,
          roleId: 1,
          role: { id: 1, orgId: 1, name: "owner", description: "Organization owner", isSystem: true },
          joinedAt: "2026-01-01T00:00:00Z",
          email: "test@example.com",
          firstName: "Test",
          lastName: "User",
        },
      ],
      error: null,
    })
  }),

  http.get(`${API_BASE}/api/orgs/:id/roles`, () => {
    return HttpResponse.json({
      data: [
        { id: 1, orgId: 1, name: "owner", description: "Organization owner", isSystem: true },
        { id: 2, orgId: 1, name: "admin", description: "Administrator", isSystem: true },
        { id: 3, orgId: 1, name: "member", description: "Member", isSystem: true },
        { id: 4, orgId: 1, name: "viewer", description: "Viewer", isSystem: true },
      ],
      error: null,
    })
  }),

  // ── Notifications ─────────────────────────────────────────
  http.get(`${API_BASE}/api/notifications/unread-count`, () => {
    return HttpResponse.json({
      data: { count: 3 },
      error: null,
    })
  }),

  http.get(`${API_BASE}/api/notifications`, () => {
    return HttpResponse.json({
      data: createNotifications(5),
      error: null,
      meta: { page: 1, limit: 20, total: 5 },
    })
  }),

  http.put(`${API_BASE}/api/notifications/:id/read`, () => {
    return HttpResponse.json({ data: null, error: null })
  }),

  // ── Notification Channels ─────────────────────────────────
  http.get(`${API_BASE}/api/orgs/:orgId/notification-channels`, () => {
    return HttpResponse.json({
      data: [createNotificationChannel()],
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/orgs/:orgId/notification-channels`, async ({ request }) => {
    const body = (await request.json()) as { name: string; type: string; config: string }
    return HttpResponse.json(
      {
        data: createNotificationChannel({
          name: body.name,
          type: body.type as "email" | "slack" | "webhook",
          config: body.config,
        }),
        error: null,
      },
      { status: 201 }
    )
  }),

  http.delete(`${API_BASE}/api/orgs/:orgId/notification-channels/:id`, () => {
    return HttpResponse.json({ data: null, error: null })
  }),

  // ── API Keys ──────────────────────────────────────────────
  http.get(`${API_BASE}/api/auth/api-keys`, () => {
    return HttpResponse.json({
      data: [
        createApiKey({ id: 1, name: "Production Key", scope: "read" }),
        createApiKey({ id: 2, name: "CI Key", scope: "write" }),
      ],
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/auth/api-keys`, async ({ request }) => {
    const body = (await request.json()) as { name: string; scope?: string }
    return HttpResponse.json(
      {
        data: {
          ...createApiKey({ name: body.name, scope: (body.scope ?? "read") as "read" | "write" | "admin" }),
          apiKey: "vmx_sk_test_1234567890abcdef",
        },
        error: null,
      },
      { status: 201 }
    )
  }),

  http.delete(`${API_BASE}/api/auth/api-keys/:id`, () => {
    return HttpResponse.json({ data: null, error: null })
  }),

  // ── Sessions ──────────────────────────────────────────────
  http.get(`${API_BASE}/api/auth/sessions`, () => {
    return HttpResponse.json({
      data: [
        createSession({ id: 1 }),
        createSession({
          id: 2,
          ipAddress: "192.168.1.100",
          userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
        }),
      ],
      error: null,
    })
  }),

  http.delete(`${API_BASE}/api/auth/sessions/:id`, () => {
    return HttpResponse.json({
      data: { message: "Session revoked" },
      error: null,
    })
  }),

  // ── Queue ─────────────────────────────────────────────────
  http.get(`${API_BASE}/api/queue/stats`, () => {
    return HttpResponse.json({
      data: createQueueStats(),
      error: null,
    })
  }),

  // ── Sync ──────────────────────────────────────────────────
  http.post(`${API_BASE}/api/sync/top-packages`, () => {
    return HttpResponse.json({
      data: { message: "Sync initiated" },
      error: null,
    })
  }),

  http.post(`${API_BASE}/api/sync/reanalyze`, () => {
    return HttpResponse.json({
      data: { message: "Reanalysis queued", queued: 10, dead_retried: 0 },
      error: null,
    })
  }),
]
