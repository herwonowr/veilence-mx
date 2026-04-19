import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type {
  Workspace,
  WorkspaceMember,
  Role,
  Permission,
  AuditLog,
  AuditLogParams,
} from "@/domains/admin/types/admin.types"

export const apiGetWorkspaces = async (): Promise<ApiResponse<Workspace[]>> =>
  fetchApi<Workspace[]>("/api/workspaces")

export const apiGetWorkspace = async (
  id: number
): Promise<ApiResponse<Workspace>> =>
  fetchApi<Workspace>(`/api/workspaces/${id}`)

export const apiCreateWorkspace = async (data: {
  name: string
  slug: string
  description?: string
}): Promise<ApiResponse<Workspace>> =>
  fetchApi<Workspace>("/api/workspaces", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiUpdateWorkspace = async (
  id: number,
  data: { name?: string; description?: string }
): Promise<ApiResponse<Workspace>> =>
  fetchApi<Workspace>(`/api/workspaces/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const apiDeleteWorkspace = async (
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${id}`, { method: "DELETE" })

export const apiGetWorkspaceMembers = async (
  workspaceId: number
): Promise<ApiResponse<WorkspaceMember[]>> =>
  fetchApi<WorkspaceMember[]>(`/api/workspaces/${workspaceId}/members`)

export const apiInviteMember = async (
  workspaceId: number,
  data: { email: string; roleId: number }
): Promise<ApiResponse<{ token: string }>> =>
  fetchApi<{ token: string }>(`/api/workspaces/${workspaceId}/invitations`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiRemoveMember = async (
  workspaceId: number,
  userId: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/members/${userId}`, {
    method: "DELETE",
  })

export const apiUpdateMemberRole = async (
  workspaceId: number,
  userId: number,
  roleId: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/members/${userId}/role`, {
    method: "PUT",
    body: JSON.stringify({ roleId }),
  })

export const apiGetWorkspaceRoles = async (
  workspaceId: number
): Promise<ApiResponse<Role[]>> =>
  fetchApi<Role[]>(`/api/workspaces/${workspaceId}/roles`)

export const apiGetPermissions = async (): Promise<ApiResponse<Permission[]>> =>
  fetchApi<Permission[]>("/api/permissions")

export const apiGetAuditLogs = async (
  workspaceId: number,
  params?: AuditLogParams
): Promise<ApiResponse<AuditLog[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.action) searchParams.set("action", params.action)
  if (params?.resource) searchParams.set("resource", params.resource)
  if (params?.from) searchParams.set("from", params.from)
  if (params?.to) searchParams.set("to", params.to)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  const query = searchParams.toString()
  return fetchApi<AuditLog[]>(
    `/api/workspaces/${workspaceId}/audit-logs${query ? `?${query}` : ""}`
  )
}
