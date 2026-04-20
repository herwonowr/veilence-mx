import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type {
  Workspace,
  WorkspaceMember,
  Role,
  Permission,
  AuditLog,
  AuditLogParams,
  Invitation,
  InvitationInfo,
  MyInvitation,
} from "@/domains/admin/types/admin.types"

export const apiGetWorkspaces = async (): Promise<ApiResponse<Workspace[]>> =>
  fetchApi<Workspace[]>("/api/workspaces")

export const apiGetWorkspace = async (
  id: string
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
  id: string,
  data: { name?: string; description?: string }
): Promise<ApiResponse<Workspace>> =>
  fetchApi<Workspace>(`/api/workspaces/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const apiDeleteWorkspace = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${id}`, { method: "DELETE" })

export const apiGetWorkspaceMembers = async (
  workspaceId: string
): Promise<ApiResponse<WorkspaceMember[]>> =>
  fetchApi<WorkspaceMember[]>(`/api/workspaces/${workspaceId}/members`)

export const apiInviteMember = async (
  workspaceId: string,
  data: { email: string; roleId: string }
): Promise<ApiResponse<{ token: string }>> =>
  fetchApi<{ token: string }>(`/api/workspaces/${workspaceId}/invitations`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiRemoveMember = async (
  workspaceId: string,
  userId: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/members/${userId}`, {
    method: "DELETE",
  })

export const apiUpdateMemberRole = async (
  workspaceId: string,
  userId: string,
  roleId: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/members/${userId}/role`, {
    method: "PUT",
    body: JSON.stringify({ roleId }),
  })

export const apiGetWorkspaceRoles = async (
  workspaceId: string
): Promise<ApiResponse<Role[]>> =>
  fetchApi<Role[]>(`/api/workspaces/${workspaceId}/roles`)

export const apiGetPermissions = async (): Promise<ApiResponse<Permission[]>> =>
  fetchApi<Permission[]>("/api/permissions")

export const apiGetPendingInvitations = async (
  workspaceId: string
): Promise<ApiResponse<Invitation[]>> =>
  fetchApi<Invitation[]>(`/api/workspaces/${workspaceId}/invitations`)

export const apiRevokeInvitation = async (
  workspaceId: string,
  invitationId: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/invitations/${invitationId}`, {
    method: "DELETE",
  })

export const apiGetAuditLogs = async (
  workspaceId: string,
  params?: AuditLogParams
): Promise<ApiResponse<AuditLog[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.action) searchParams.set("action", params.action)
  if (params?.resource) searchParams.set("resource", params.resource)
  if (params?.from_date) searchParams.set("from_date", params.from_date)
  if (params?.to_date) searchParams.set("to_date", params.to_date)
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  const query = searchParams.toString()
  return fetchApi<AuditLog[]>(
    `/api/workspaces/${workspaceId}/audit-logs${query ? `?${query}` : ""}`
  )
}

export const apiGetInvitationByToken = async (
  token: string
): Promise<ApiResponse<InvitationInfo>> =>
  fetchApi<InvitationInfo>(`/api/invitations/${token}`, { skipAuth: true })

export const apiResendInvitation = async (
  workspaceId: string,
  invitationId: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/invitations/${invitationId}/resend`, {
    method: "POST",
  })

export const apiAcceptInvitation = async (
  workspaceId: string,
  token: string
): Promise<ApiResponse<WorkspaceMember>> =>
  fetchApi<WorkspaceMember>(
    `/api/workspaces/${workspaceId}/invitations/${token}/accept`,
    { method: "POST" }
  )

export const apiDeclineInvitationByToken = async (
  workspaceId: string,
  token: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(
    `/api/workspaces/${workspaceId}/invitations/${token}/decline`,
    { method: "POST" }
  )

export const apiGetMyInvitations = async (): Promise<ApiResponse<MyInvitation[]>> =>
  fetchApi<MyInvitation[]>("/api/invitations/mine")

export const apiAcceptInvitationById = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/invitations/${id}/accept`, { method: "POST" })

export const apiDeclineInvitationById = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/invitations/${id}/decline`, { method: "POST" })
