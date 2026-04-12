/**
 * Shared test fixtures and mock factories for Veilence-MX frontend tests.
 *
 * Usage:
 *   import { fixtures } from "@/test-fixtures"
 *   const user = fixtures.user()
 *   const org = fixtures.org({ name: "Custom Org" })
 *   const alerts = fixtures.alerts(5) // array of 5 alerts
 */

import type {
  User,
  Organization,
  Package,
  Release,
  Alert,
  DashboardStats,
  ChartData,
  Notification,
  ApiKeyInfo,
  Session,
  QueueStatsResponse,
  RecentRelease,
  NotificationChannel,
} from "@/types"

// ─── User Fixtures ──────────────────────────────────────────────

export function createUser(overrides: Partial<User> = {}): User {
  return {
    id: 1,
    email: "test@example.com",
    firstName: "Test",
    lastName: "User",
    isActive: true,
    emailVerified: true,
    lastLoginAt: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

// ─── Organization Fixtures ──────────────────────────────────────

export function createOrg(overrides: Partial<Organization> = {}): Organization {
  return {
    id: 1,
    name: "Test Org",
    slug: "test-org",
    description: "Test organization",
    ownerId: 1,
    isActive: true,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

// ─── Package Fixtures ───────────────────────────────────────────

export function createPackage(overrides: Partial<Package> = {}): Package {
  return {
    id: 1,
    name: "test-package",
    ecosystem: "npm",
    latestVersion: "1.0.0",
    description: "A test package",
    isCustom: false,
    rank: null,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

export function createPackages(count: number, base: Partial<Package> = {}): Package[] {
  return Array.from({ length: count }, (_, i) =>
    createPackage({
      id: i + 1,
      name: `package-${i + 1}`,
      ...base,
    })
  )
}

// ─── Release Fixtures ───────────────────────────────────────────

export function createRelease(overrides: Partial<Release> = {}): Release {
  return {
    id: 1,
    packageId: 1,
    version: "1.0.0",
    publishedAt: "2026-01-01T00:00:00Z",
    tarballUrl: "https://registry.npmjs.org/test-package/-/test-package-1.0.0.tgz",
    sha256: "abc123def456",
    status: "completed",
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

export function createRecentRelease(overrides: Partial<RecentRelease> = {}): RecentRelease {
  return {
    ...createRelease(),
    packageName: "test-package",
    packageEcosystem: "npm",
    classification: "benign",
    ...overrides,
  }
}

// ─── Alert Fixtures ─────────────────────────────────────────────

export function createAlert(overrides: Partial<Alert> = {}): Alert {
  return {
    id: 1,
    analysisId: 1,
    packageId: 1,
    severity: "high",
    status: "new",
    message: "Suspicious code pattern detected",
    packageName: "test-package",
    packageEcosystem: "npm",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

export function createAlerts(count: number, base: Partial<Alert> = {}): Alert[] {
  const severities: Alert["severity"][] = ["low", "medium", "high", "critical"]
  return Array.from({ length: count }, (_, i) =>
    createAlert({
      id: i + 1,
      severity: severities[i % severities.length],
      ...base,
    })
  )
}

// ─── Dashboard Fixtures ─────────────────────────────────────────

export function createDashboardStats(overrides: Partial<DashboardStats> = {}): DashboardStats {
  return {
    totalPackages: 150,
    totalReleases: 1200,
    pendingAnalyses: 5,
    activeAlerts: 3,
    recentMalicious: 1,
    ...overrides,
  }
}

export function createChartData(overrides: Partial<ChartData> = {}): ChartData {
  return {
    releaseActivity: [
      { date: "2026-03-30", releases: 10 },
      { date: "2026-03-31", releases: 15 },
      { date: "2026-04-01", releases: 8 },
    ],
    classifications: [
      { classification: "benign", count: 100 },
      { classification: "suspicious", count: 20 },
      { classification: "malicious", count: 3 },
    ],
    ecosystems: [
      { ecosystem: "npm", count: 80 },
      { ecosystem: "python", count: 70 },
    ],
    alertsBySeverity: [
      { severity: "low", count: 5 },
      { severity: "medium", count: 3 },
      { severity: "high", count: 2 },
      { severity: "critical", count: 1 },
    ],
    releaseStatuses: [
      { status: "completed", count: 100 },
      { status: "pending", count: 5 },
      { status: "error", count: 2 },
    ],
    ...overrides,
  }
}

// ─── Notification Fixtures ──────────────────────────────────────

export function createNotification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: 1,
    orgId: 1,
    userId: 1,
    channelId: 1,
    title: "New Alert",
    message: "A critical alert was raised for package evil-lib",
    isRead: false,
    sentAt: "2026-01-01T00:00:00Z",
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

export function createNotifications(count: number, base: Partial<Notification> = {}): Notification[] {
  return Array.from({ length: count }, (_, i) =>
    createNotification({
      id: i + 1,
      title: `Notification ${i + 1}`,
      isRead: i > count / 2, // half read, half unread
      ...base,
    })
  )
}

export function createNotificationChannel(
  overrides: Partial<NotificationChannel> = {}
): NotificationChannel {
  return {
    id: 1,
    orgId: 1,
    name: "Slack #alerts",
    type: "slack",
    config: JSON.stringify({ webhookUrl: "https://hooks.slack.com/test" }),
    isActive: true,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

// ─── API Key Fixtures ───────────────────────────────────────────

export function createApiKey(overrides: Partial<ApiKeyInfo> = {}): ApiKeyInfo {
  return {
    id: 1,
    userId: 1,
    name: "Test API Key",
    keyPrefix: "vmx_sk_test",
    scope: "read",
    lastUsedAt: null,
    expiresAt: null,
    isActive: true,
    createdAt: "2026-01-01T00:00:00Z",
    ...overrides,
  }
}

// ─── Session Fixtures ───────────────────────────────────────────

export function createSession(overrides: Partial<Session> = {}): Session {
  return {
    id: 1,
    userId: 1,
    ipAddress: "127.0.0.1",
    userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    createdAt: "2026-01-01T00:00:00Z",
    lastActive: "2026-04-06T00:00:00Z",
    expiresAt: "2026-04-07T00:00:00Z",
    ...overrides,
  }
}

// ─── Queue Fixtures ─────────────────────────────────────────────

export function createQueueStats(
  overrides: Partial<QueueStatsResponse> = {}
): QueueStatsResponse {
  return {
    diff: { pending: 5, processing: 2, completed: 100, failed: 1, dead: 0 },
    analyze: { pending: 3, processing: 1, completed: 80, failed: 0, dead: 0 },
    ...overrides,
  }
}

// ─── Auth Mock Helpers ──────────────────────────────────────────

/**
 * Creates a full mock return value for useAuth().
 * Use with: vi.mocked(useAuth).mockReturnValue(createAuthState())
 */
export function createAuthState(overrides: {
  user?: User | null
  isAuthenticated?: boolean
  isLoading?: boolean
  currentOrg?: Organization | null
  organizations?: Organization[]
} = {}) {
  const user = overrides.user !== undefined ? overrides.user : createUser()
  return {
    user,
    isAuthenticated: overrides.isAuthenticated ?? (user !== null),
    isLoading: overrides.isLoading ?? false,
    currentOrg: overrides.currentOrg !== undefined ? overrides.currentOrg : createOrg(),
    organizations: overrides.organizations ?? [createOrg()],
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    setCurrentOrg: vi.fn(),
    refreshUser: vi.fn(),
    refreshOrgs: vi.fn(),
  }
}

/**
 * Creates unauthenticated auth state.
 * Use with: vi.mocked(useAuth).mockReturnValue(createUnauthState())
 */
export function createUnauthState() {
  return createAuthState({
    user: null,
    isAuthenticated: false,
    currentOrg: null,
    organizations: [],
  })
}

// ─── API Response Helpers ───────────────────────────────────────

/**
 * Wraps data in the standard API response envelope.
 */
export function apiResponse<T>(data: T, meta?: { page: number; limit: number; total: number }) {
  return {
    data,
    error: null,
    ...(meta ? { meta } : {}),
  }
}

/**
 * Creates an API error response.
 */
export function apiError(error: string) {
  return {
    data: null,
    error,
  }
}

// ─── Convenience namespace ──────────────────────────────────────

export const fixtures = {
  user: createUser,
  org: createOrg,
  package: createPackage,
  packages: createPackages,
  release: createRelease,
  recentRelease: createRecentRelease,
  alert: createAlert,
  alerts: createAlerts,
  dashboardStats: createDashboardStats,
  chartData: createChartData,
  notification: createNotification,
  notifications: createNotifications,
  notificationChannel: createNotificationChannel,
  apiKey: createApiKey,
  session: createSession,
  queueStats: createQueueStats,
  authState: createAuthState,
  unauthState: createUnauthState,
  apiResponse,
  apiError,
} as const
