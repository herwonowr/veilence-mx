import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  apiGetOrgs,
  apiGetOrg,
  apiCreateOrg,
  apiUpdateOrg,
  apiDeleteOrg,
  apiGetOrgMembers,
  apiGetOrgRoles,
  apiGetPermissions,
  apiInviteMember,
  apiRemoveMember,
  apiUpdateMemberRole,
  apiGetAuditLogs,
} from "@/lib/api-client"
import type {
  ApiResponse,
  Organization,
  OrgMember,
  Role,
  Permission,
  AuditLog,
} from "@/types"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/lib/utils"

export const orgKeys = {
  all: ["organizations"] as const,
  lists: () => [...orgKeys.all, "list"] as const,
  detail: (id: number) => [...orgKeys.all, "detail", id] as const,
  members: (orgId: number) => [...orgKeys.all, "members", orgId] as const,
  roles: (orgId: number) => [...orgKeys.all, "roles", orgId] as const,
  permissions: () => [...orgKeys.all, "permissions"] as const,
  auditLogs: (orgId: number, params?: Record<string, unknown>) =>
    [...orgKeys.all, "audit-logs", orgId, params] as const,
}

export function useOrganizations(
  options?: Partial<UseQueryOptions<ApiResponse<Organization[]>>>
) {
  return useQuery({
    queryKey: orgKeys.lists(),
    queryFn: () => apiGetOrgs(),
    ...options,
  })
}

export function useOrganization(
  id: number,
  options?: Partial<UseQueryOptions<ApiResponse<Organization>>>
) {
  return useQuery({
    queryKey: orgKeys.detail(id),
    queryFn: () => apiGetOrg(id),
    enabled: id > 0,
    ...options,
  })
}

export function useCreateOrganization() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { name: string; slug: string; description?: string }) =>
      apiCreateOrg(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.lists() })
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to create organization"))
    },
  })
}

export function useUpdateOrganization() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: number
      data: { name?: string; description?: string }
    }) => apiUpdateOrg(id, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({ queryKey: orgKeys.detail(variables.id) })
      queryClient.invalidateQueries({ queryKey: orgKeys.lists() })
      toast.success("Organization updated")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to update organization"))
    },
  })
}

export function useDeleteOrganization() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiDeleteOrg(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.lists() })
      toast.success("Organization deleted")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to delete organization"))
    },
  })
}

export function useOrgMembers(
  orgId: number,
  options?: Partial<UseQueryOptions<ApiResponse<OrgMember[]>>>
) {
  return useQuery({
    queryKey: orgKeys.members(orgId),
    queryFn: () => apiGetOrgMembers(orgId),
    enabled: orgId > 0,
    ...options,
  })
}

export function useOrgRoles(
  orgId: number,
  options?: Partial<UseQueryOptions<ApiResponse<Role[]>>>
) {
  return useQuery({
    queryKey: orgKeys.roles(orgId),
    queryFn: () => apiGetOrgRoles(orgId),
    enabled: orgId > 0,
    ...options,
  })
}

export function usePermissions(
  options?: Partial<UseQueryOptions<ApiResponse<Permission[]>>>
) {
  return useQuery({
    queryKey: orgKeys.permissions(),
    queryFn: () => apiGetPermissions(),
    ...options,
  })
}

export function useInviteMember() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      orgId,
      data,
    }: {
      orgId: number
      data: { email: string; roleId: number }
    }) => apiInviteMember(orgId, data),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: orgKeys.members(variables.orgId),
      })
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to send invitation"))
    },
  })
}

export function useRemoveMember() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ orgId, userId }: { orgId: number; userId: number }) =>
      apiRemoveMember(orgId, userId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: orgKeys.members(variables.orgId),
      })
      toast.success("Member removed")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to remove member"))
    },
  })
}

export function useUpdateMemberRole() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      orgId,
      userId,
      roleId,
    }: {
      orgId: number
      userId: number
      roleId: number
    }) => apiUpdateMemberRole(orgId, userId, roleId),
    onSuccess: (_result, variables) => {
      queryClient.invalidateQueries({
        queryKey: orgKeys.members(variables.orgId),
      })
      toast.success("Member role updated")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to update role"))
    },
  })
}

export function useAuditLogs(
  orgId: number,
  params?: {
    action?: string
    resource?: string
    from?: string
    to?: string
    page?: number
    limit?: number
  },
  options?: Partial<UseQueryOptions<ApiResponse<AuditLog[]>>>
) {
  return useQuery({
    queryKey: orgKeys.auditLogs(orgId, params as Record<string, unknown>),
    queryFn: () => apiGetAuditLogs(orgId, params),
    enabled: orgId > 0,
    ...options,
  })
}
