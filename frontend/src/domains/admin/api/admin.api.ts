import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type {
  Organization,
  OrgMember,
  Role,
  Permission,
  AuditLog,
  AuditLogParams,
} from "@/domains/admin/types/admin.types"

export const apiGetOrgs = async (): Promise<ApiResponse<Organization[]>> =>
  fetchApi<Organization[]>("/api/orgs")

export const apiGetOrg = async (
  id: number
): Promise<ApiResponse<Organization>> =>
  fetchApi<Organization>(`/api/orgs/${id}`)

export const apiCreateOrg = async (data: {
  name: string
  slug: string
  description?: string
}): Promise<ApiResponse<Organization>> =>
  fetchApi<Organization>("/api/orgs", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiUpdateOrg = async (
  id: number,
  data: { name?: string; description?: string }
): Promise<ApiResponse<Organization>> =>
  fetchApi<Organization>(`/api/orgs/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const apiDeleteOrg = async (
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/orgs/${id}`, { method: "DELETE" })

export const apiGetOrgMembers = async (
  orgId: number
): Promise<ApiResponse<OrgMember[]>> =>
  fetchApi<OrgMember[]>(`/api/orgs/${orgId}/members`)

export const apiInviteMember = async (
  orgId: number,
  data: { email: string; roleId: number }
): Promise<ApiResponse<{ token: string }>> =>
  fetchApi<{ token: string }>(`/api/orgs/${orgId}/invitations`, {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiRemoveMember = async (
  orgId: number,
  userId: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/orgs/${orgId}/members/${userId}`, {
    method: "DELETE",
  })

export const apiUpdateMemberRole = async (
  orgId: number,
  userId: number,
  roleId: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/orgs/${orgId}/members/${userId}/role`, {
    method: "PUT",
    body: JSON.stringify({ roleId }),
  })

export const apiGetOrgRoles = async (
  orgId: number
): Promise<ApiResponse<Role[]>> =>
  fetchApi<Role[]>(`/api/orgs/${orgId}/roles`)

export const apiGetPermissions = async (): Promise<ApiResponse<Permission[]>> =>
  fetchApi<Permission[]>("/api/permissions")

export const apiGetAuditLogs = async (
  orgId: number,
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
    `/api/orgs/${orgId}/audit-logs${query ? `?${query}` : ""}`
  )
}
