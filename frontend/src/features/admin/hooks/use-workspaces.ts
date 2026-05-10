"use client"

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  apiGetWorkspaces,
  apiGetWorkspace,
  apiUpdateWorkspace,
  apiDeleteWorkspace,
  apiGetWorkspaceMembers,
  apiGetWorkspaceRoles,
  apiInviteMember,
  apiRemoveMember,
  apiUpdateMemberRole,
  apiGetAuditLogs,
  apiGetPendingInvitations,
  apiRevokeInvitation,
  apiResendInvitation,
  apiAddMember,
} from "@/domains/admin"
import type { ApiResponse } from "@/domains/common"
import type {
  Workspace,
  WorkspaceMember,
  Role,
  AuditLog,
  Invitation,
} from "@/domains/admin"
import type { AddMemberRequest } from "@/domains/admin"
import { toast } from "sonner"
import { sanitizeErrorMessage , usePublicConfigQuery } from "@/core"

export const workspaceKeys = {
  all: ["workspaces"] as const,
  listsBase: () => [...workspaceKeys.all, "list"] as const,
  lists: (params?: { search?: string; page?: number; limit?: number }) =>
    [...workspaceKeys.listsBase(), params] as const,
  detail: (id: string) => [...workspaceKeys.all, "detail", id] as const,
  members: (workspaceId: string) => [...workspaceKeys.all, "members", workspaceId] as const,
  invitations: (workspaceId: string) => [...workspaceKeys.all, "invitations", workspaceId] as const,
  roles: (workspaceId: string) => [...workspaceKeys.all, "roles", workspaceId] as const,
  permissions: () => [...workspaceKeys.all, "permissions"] as const,
  auditLogs: (workspaceId: string, params?: Record<string, unknown>) =>
    [...workspaceKeys.all, "audit-logs", workspaceId, params] as const,
}

export const useWorkspaces = (
  params?: { search?: string; page?: number; limit?: number },
  options?: Partial<UseQueryOptions<ApiResponse<Workspace[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.lists(params),
    queryFn: () => apiGetWorkspaces(params),
    ...options,
  })
}

export const useWorkspace = (
  id: string,
  options?: Partial<UseQueryOptions<ApiResponse<Workspace>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.detail(id),
    queryFn: () => apiGetWorkspace(id),
    enabled: !!id,
    ...options,
  })
}

export const useUpdateWorkspace = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: { name?: string; description?: string }
    }) => apiUpdateWorkspace(id, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(variables.id) })
      queryClient.invalidateQueries({ queryKey: workspaceKeys.listsBase() })
      toast.success("Workspace updated")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to update workspace"))
    },
  })
}

export const useDeleteWorkspace = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiDeleteWorkspace(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.listsBase() })
      toast.success("Workspace deleted")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to delete workspace"))
    },
  })
}

export const useWorkspaceMembers = (
  workspaceId: string,
  options?: Partial<UseQueryOptions<ApiResponse<WorkspaceMember[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.members(workspaceId),
    queryFn: () => apiGetWorkspaceMembers(workspaceId),
    enabled: !!workspaceId,
    ...options,
  })
}

export const useWorkspaceRoles = (
  workspaceId: string,
  options?: Partial<UseQueryOptions<ApiResponse<Role[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.roles(workspaceId),
    queryFn: () => apiGetWorkspaceRoles(workspaceId),
    enabled: !!workspaceId,
    ...options,
  })
}

export const useInviteMember = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      workspaceId,
      data,
    }: {
      workspaceId: string
      data: { email: string; roleId: string }
    }) => apiInviteMember(workspaceId, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.members(variables.workspaceId),
      })
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.invitations(variables.workspaceId),
      })
      toast.success("Invitation sent")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to send invitation"))
    },
  })
}

export const usePendingInvitations = (
  workspaceId: string,
  options?: Partial<UseQueryOptions<ApiResponse<Invitation[]>>>
) => {
  const { registrationEnabled } = usePublicConfigQuery()

  return useQuery({
    queryKey: workspaceKeys.invitations(workspaceId),
    queryFn: () => apiGetPendingInvitations(workspaceId),
    enabled: !!workspaceId && registrationEnabled,
    ...options,
  })
}

export const useRevokeInvitation = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      workspaceId,
      invitationId,
    }: {
      workspaceId: string
      invitationId: string
    }) => apiRevokeInvitation(workspaceId, invitationId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.invitations(variables.workspaceId),
      })
      toast.success("Invitation revoked")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to revoke invitation"))
    },
  })
}

export const useResendInvitation = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      workspaceId,
      invitationId,
    }: {
      workspaceId: string
      invitationId: string
    }) => apiResendInvitation(workspaceId, invitationId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.invitations(variables.workspaceId),
      })
      toast.success("Invitation resent")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to resend invitation"))
    },
  })
}

export const useRemoveMember = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ workspaceId, userId }: { workspaceId: string; userId: string }) =>
      apiRemoveMember(workspaceId, userId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.members(variables.workspaceId),
      })
      toast.success("Member removed")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to remove member"))
    },
  })
}

export const useUpdateMemberRole = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      workspaceId,
      userId,
      roleId,
    }: {
      workspaceId: string
      userId: string
      roleId: string
    }) => apiUpdateMemberRole(workspaceId, userId, roleId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.members(variables.workspaceId),
      })
      toast.success("Member role updated")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to update role"))
    },
  })
}

export const useAuditLogs = (
  workspaceId: string,
  params?: {
    action?: string
    resource?: string
    from_date?: string
    to_date?: string
    page?: number
    limit?: number
    sort_by?: string
    sort_dir?: "asc" | "desc"
  },
  options?: Partial<UseQueryOptions<ApiResponse<AuditLog[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.auditLogs(workspaceId, params as Record<string, unknown>),
    queryFn: () => apiGetAuditLogs(workspaceId, params),
    enabled: !!workspaceId,
    ...options,
  })
}

export const useAddMember = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      workspaceId,
      data,
    }: {
      workspaceId: string
      data: AddMemberRequest
    }) => apiAddMember(workspaceId, data),
    onSuccess: (result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.members(variables.workspaceId),
      })
      const response = result.data
      if (response?.userCreated) {
        if (variables.data.password) {
          toast.success("User added. They'll need to change their password on first login.")
        } else {
          toast.success("User added. They'll receive an email to set their password.")
        }
      } else {
        toast.success("User added to workspace.")
      }
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to add member"))
    },
  })
}
