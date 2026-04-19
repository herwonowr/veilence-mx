"use client"

/**
 * Re-export from core/hooks/use-workspace-role so that features/auth barrel
 * consumers don't need to change their imports.
 */
export {
  useCurrentWorkspaceRole,
  hasMinimumRole,
  getRoleLevel,
  workspaceRoleKeys,
  type WorkspaceRole,
} from "@/core/hooks/use-workspace-role"
