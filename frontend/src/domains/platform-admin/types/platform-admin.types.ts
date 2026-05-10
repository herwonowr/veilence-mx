export interface PlatformAuthSettings {
  passwordLoginEnabled: boolean
  registrationEnabled: boolean
}

export interface UpdatePlatformAuthSettingsRequest {
  passwordLoginEnabled?: boolean
}

export interface PlatformUserSummary {
  id: string
  email: string
  firstName: string
  lastName: string
  isSuperAdmin: boolean
  isActive: boolean
  emailVerified: boolean
  mustChangePassword: boolean
  deactivatedAt: string | null
  authMethod: "password" | "google" | "github" | "saml"
  createdAt: string
  updatedAt: string
  lastLoginAt: string | null
}

export interface PlatformUserDetail extends PlatformUserSummary {
  identities: LinkedIdentity[]
  workspaces: UserWorkspace[]
}

interface LinkedIdentity {
  id: string
  provider: string
  providerEmail: string
  providerUserId: string
}

interface UserWorkspace {
  id: string
  name: string
  role: string
}

export interface UpdatePlatformUserRequest {
  firstName?: string
  lastName?: string
  isSuperAdmin?: boolean
  isActive?: boolean
}

export interface ListUsersResponse {
  users: PlatformUserSummary[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export interface PlatformAuditLog {
  id: string
  userId: string
  userEmail: string
  workspaceId: string
  workspaceName: string
  action: string
  resource: string
  resourceId: string
  details: string
  ipAddress: string
  userAgent: string
  correlationId: string
  createdAt: string
}

export interface PlatformAuditLogParams {
  page?: number
  limit?: number
  action?: string
  resource?: string
  user_email?: string
  workspace_name?: string
  from_date?: string
  to_date?: string
  sort_by?: string
  sort_dir?: string
}
