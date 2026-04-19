"use client"

export {
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
} from "@/features/admin/hooks/use-workspaces"

export {
  useAdminPackages,
  adminPackageKeys,
} from "@/features/admin/hooks/use-admin-packages"

export { WorkspaceSelector } from "@/features/admin/ui/workspace-selector"
export { WorkspacesListView } from "@/features/admin/ui/workspaces-list-view"
export { WorkspaceDetailView } from "@/features/admin/ui/workspace-detail-view"
export { AuditLogView } from "@/features/admin/ui/audit-log-view"
