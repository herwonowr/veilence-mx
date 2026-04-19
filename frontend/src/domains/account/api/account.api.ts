import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { ApiKeyInfo, ApiKeyCreatedResponse, CreateApiKeyRequest } from "@/domains/account/types/account.types"

export const apiGetApiKeys = async (): Promise<ApiResponse<ApiKeyInfo[]>> =>
  fetchApi<ApiKeyInfo[]>("/api/auth/api-keys")

export const apiCreateApiKey = async (
  data: CreateApiKeyRequest
): Promise<ApiResponse<ApiKeyCreatedResponse>> =>
  fetchApi<ApiKeyCreatedResponse>("/api/auth/api-keys", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiDeleteApiKey = async (
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/auth/api-keys/${id}`, { method: "DELETE" })

export const apiGetCurrentMemberRole = async (
  workspaceId: number
): Promise<ApiResponse<{ role: string }>> =>
  fetchApi<{ role: string }>(`/api/workspaces/${workspaceId}/members/me/role`)
