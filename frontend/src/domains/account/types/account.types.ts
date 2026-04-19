export type APIKeyRole = "owner" | "admin" | "member" | "viewer"

export const API_KEY_ROLE_HIERARCHY: APIKeyRole[] = ["owner", "admin", "member", "viewer"]

export interface ApiKeyInfo {
  id: number
  userId: number
  workspaceId: number
  name: string
  keyPrefix: string
  role: APIKeyRole
  lastUsedAt: string | null
  expiresAt: string | null
  isActive: boolean
  createdAt: string
}

export interface CreateApiKeyRequest {
  name: string
  role?: APIKeyRole
  expiresAt?: string
}

export interface ApiKeyCreatedResponse {
  apiKey: ApiKeyInfo
  key: string
}
