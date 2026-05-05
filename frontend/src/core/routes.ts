/**
 * Centralized route path constants for the entire frontend.
 *
 * This file is pure TypeScript (no React) so it can be imported by
 * server-side code like proxy.ts as well as client components.
 */

export const ROUTES = {
  // Auth pages
  LOGIN: "/login",
  REGISTER: "/register",
  FORGOT_PASSWORD: "/forgot-password",
  RESET_PASSWORD: "/reset-password",
  VERIFY_EMAIL: "/verify-email",
  SETUP: "/setup",
  CHANGE_PASSWORD: "/change-password",
  INVITE: "/invite",
  SSO_CALLBACK: "/auth/sso/callback",

  // App pages
  DASHBOARD: "/",
  PACKAGES: "/packages",
  PACKAGE_DETAIL: (id: string) => `/packages/${id}`,
  PACKAGES_STALE: "/packages/stale",
  PACKAGES_SUGGESTIONS: "/packages/suggestions",
  PACKAGES_IMPORT: "/packages/import",
  RELEASES: "/releases",
  RELEASE_DETAIL: (id: string) => `/releases/${id}`,
  ALERTS: "/alerts",
  ALERT_DETAIL: (id: string) => `/alerts/${id}`,
  WORKSPACES: "/workspaces",
  WORKSPACE_DETAIL: (id: string) => `/workspaces/${id}`,
  WORKSPACE_AUDIT: (id: string) => `/workspaces/${id}/audit`,
  WORKSPACES_INVITATIONS: "/workspaces/invitations",
  SETTINGS: "/settings",
  SETTINGS_API_KEYS: "/settings/api-keys",
  SETTINGS_SESSIONS: "/settings/sessions",
  SETTINGS_QUEUE: "/settings/queue",
  SETTINGS_NOTIFICATIONS: "/settings/notifications",
  ADMIN_SECURITY: "/admin/security",
  ADMIN_USERS: "/admin/users",
  ADMIN_AUDIT_LOGS: "/admin/audit-logs",
  ACCOUNT: "/account",
  NOTIFICATIONS: "/notifications",
} as const

/** Routes accessible without authentication */
export const PUBLIC_PATHS = [
  ROUTES.LOGIN,
  ROUTES.REGISTER,
  ROUTES.FORGOT_PASSWORD,
  ROUTES.RESET_PASSWORD,
  ROUTES.VERIFY_EMAIL,
  ROUTES.SETUP,
  ROUTES.INVITE,
  ROUTES.SSO_CALLBACK,
] as const

/** Auth pages where authenticated users should be redirected to dashboard */
export const AUTH_PAGE_PATHS = [
  ROUTES.LOGIN,
  ROUTES.REGISTER,
  ROUTES.FORGOT_PASSWORD,
  ROUTES.RESET_PASSWORD,
] as const

/** Pages that require super-admin privileges */
export const SUPER_ADMIN_PATHS = [
  ROUTES.ADMIN_SECURITY,
  ROUTES.ADMIN_USERS,
  ROUTES.ADMIN_AUDIT_LOGS,
] as const

/** Pages that render without app shell (no sidebar/header) */
export const NO_CHROME_PATHS = [
  ROUTES.LOGIN,
  ROUTES.REGISTER,
  ROUTES.FORGOT_PASSWORD,
  ROUTES.RESET_PASSWORD,
  ROUTES.VERIFY_EMAIL,
  ROUTES.SETUP,
  ROUTES.CHANGE_PASSWORD,
  ROUTES.INVITE,
] as const
