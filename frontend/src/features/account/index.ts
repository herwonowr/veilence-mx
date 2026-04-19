"use client"

export {
  useApiKeys,
  useCreateApiKey,
  useDeleteApiKey,
  apiKeyKeys,
} from "@/features/account/hooks/use-api-keys"

/**
 * useCurrentWorkspaceRole has been moved to features/auth for shared access.
 * Re-export here for backward compatibility.
 */
export { useCurrentWorkspaceRole } from "@/core/hooks/use-workspace-role"

export {
  useUpdateProfile,
  useChangePassword,
  useSendVerification,
} from "@/features/account/hooks/use-profile"

export {
  useSessions,
  useRevokeSession,
  sessionKeys,
} from "@/features/account/hooks/use-sessions"

export { AccountView } from "@/features/account/ui/account-view"
export { ApiKeysView } from "@/features/account/ui/api-keys-view"
export { SessionsView } from "@/features/account/ui/sessions-view"
