export type APIKeyScope = "read" | "write" | "admin"

export interface ApiKeyInfo {
  id: number
  userId: number
  name: string
  keyPrefix: string
  scope: APIKeyScope
  lastUsedAt: string | null
  expiresAt: string | null
  isActive: boolean
  createdAt: string
}

export interface CreateApiKeyRequest {
  name: string
  scope?: APIKeyScope
  expiresAt?: string
}

export interface ApiKeyCreatedResponse {
  apiKey: ApiKeyInfo
  key: string
}
