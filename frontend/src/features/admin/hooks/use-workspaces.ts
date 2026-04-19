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
  apiCreateWorkspace,
  apiUpdateWorkspace,
  apiDeleteWorkspace,
  apiGetWorkspaceMembers,
  apiGetWorkspaceRoles,
  apiGetPermissions,
  apiInviteMember,
  apiRemoveMember,
  apiUpdateMemberRole,
  apiGetAuditLogs,
} from "@/domains/admin"
import type { ApiResponse } from "@/domains/common"
import type {
  Workspace,
  WorkspaceMember,
  Role,
  Permission,
  AuditLog,
} from "@/domains/admin"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const workspaceKeys = {
  all: ["workspaces"] as const,
  lists: () => [...workspaceKeys.all, "list"] as const,
  detail: (id: number) => [...workspaceKeys.all, "detail", id] as const,
  members: (workspaceId: number) => [...workspaceKeys.all, "members", workspaceId] as const,
  roles: (workspaceId: number) => [...workspaceKeys.all, "roles", workspaceId] as const,
  permissions: () => [...workspaceKeys.all, "permissions"] as const,
  auditLogs: (workspaceId: number, params?: Record<string, unknown>) =>
    [...workspaceKeys.all, "audit-logs", workspaceId, params] as const,
}

export const useWorkspaces = (
  options?: Partial<UseQueryOptions<ApiResponse<Workspace[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.lists(),
    queryFn: () => apiGetWorkspaces(),
    ...options,
  })
}

export const useWorkspace = (
  id: number,
  options?: Partial<UseQueryOptions<ApiResponse<Workspace>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.detail(id),
    queryFn: () => apiGetWorkspace(id),
    enabled: id > 0,
    ...options,
  })
}

export const useCreateWorkspace = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { name: string; slug: string; description?: string }) =>
      apiCreateWorkspace(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() })
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to create workspace"))
    },
  })
}

export const useUpdateWorkspace = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: number
      data: { name?: string; description?: string }
    }) => apiUpdateWorkspace(id, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(variables.id) })
      queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() })
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
    mutationFn: (id: number) => apiDeleteWorkspace(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() })
      toast.success("Workspace deleted")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to delete workspace"))
    },
  })
}

export const useWorkspaceMembers = (
  workspaceId: number,
  options?: Partial<UseQueryOptions<ApiResponse<WorkspaceMember[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.members(workspaceId),
    queryFn: () => apiGetWorkspaceMembers(workspaceId),
    enabled: workspaceId > 0,
    ...options,
  })
}

export const useWorkspaceRoles = (
  workspaceId: number,
  options?: Partial<UseQueryOptions<ApiResponse<Role[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.roles(workspaceId),
    queryFn: () => apiGetWorkspaceRoles(workspaceId),
    enabled: workspaceId > 0,
    ...options,
  })
}

export const usePermissions = (
  options?: Partial<UseQueryOptions<ApiResponse<Permission[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.permissions(),
    queryFn: () => apiGetPermissions(),
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
      workspaceId: number
      data: { email: string; roleId: number }
    }) => apiInviteMember(workspaceId, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: workspaceKeys.members(variables.workspaceId),
      })
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to send invitation"))
    },
  })
}

export const useRemoveMember = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ workspaceId, userId }: { workspaceId: number; userId: number }) =>
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
      workspaceId: number
      userId: number
      roleId: number
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
  workspaceId: number,
  params?: {
    action?: string
    resource?: string
    from?: string
    to?: string
    page?: number
    limit?: number
  },
  options?: Partial<UseQueryOptions<ApiResponse<AuditLog[]>>>
) => {
  return useQuery({
    queryKey: workspaceKeys.auditLogs(workspaceId, params as Record<string, unknown>),
    queryFn: () => apiGetAuditLogs(workspaceId, params),
    enabled: workspaceId > 0,
    ...options,
  })
}
