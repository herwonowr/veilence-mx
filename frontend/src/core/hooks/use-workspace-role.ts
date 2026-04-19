"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchApi } from "@/core/http"
import { useAuth } from "@/core/providers/auth-provider"

/**
 * Shared workspace role hook — used by RequireRole, sidebar, and feature views.
 *
 * Lives in core/ so any feature can import it without cross-feature violations.
 * The role is always fetched from the server; it cannot be spoofed via
 * localStorage. Frontend role checks are UI convenience only — the backend
 * enforces authorization on every endpoint.
 */

export type WorkspaceRole = "owner" | "admin" | "member" | "viewer"

export const workspaceRoleKeys = {
  all: ["workspace-role"] as const,
  current: (workspaceId: number) =>
    [...workspaceRoleKeys.all, "current", workspaceId] as const,
}

/** Role hierarchy — higher index = more privilege */
const ROLE_HIERARCHY: readonly WorkspaceRole[] = [
  "viewer",
  "member",
  "admin",
  "owner",
] as const

export const getRoleLevel = (role: WorkspaceRole): number =>
  ROLE_HIERARCHY.indexOf(role)

export const hasMinimumRole = (
  userRole: WorkspaceRole | null,
  minimumRole: WorkspaceRole,
): boolean => {
  if (!userRole) return false
  return getRoleLevel(userRole) >= getRoleLevel(minimumRole)
}

export const useCurrentWorkspaceRole = (): {
  role: WorkspaceRole | null
  isLoading: boolean
} => {
  const { currentWorkspace } = useAuth()

  const { data, isLoading } = useQuery({
    queryKey: workspaceRoleKeys.current(currentWorkspace?.id ?? 0),
    queryFn: () =>
      fetchApi<{ role: string }>(
        `/api/workspaces/${currentWorkspace!.id}/members/me/role`,
      ),
    enabled: !!currentWorkspace,
  })

  const roleName = data?.data?.role?.toLowerCase()
  if (
    roleName === "owner" ||
    roleName === "admin" ||
    roleName === "member" ||
    roleName === "viewer"
  ) {
    return { role: roleName, isLoading }
  }
  return { role: null, isLoading }
}
