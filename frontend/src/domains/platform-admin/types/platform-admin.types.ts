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
  authMethod: "password" | "google" | "github" | "saml"
  createdAt: string
  lastLoginAt: string | null
}

export interface PlatformUserDetail extends PlatformUserSummary {
  emailVerified: boolean
  identities: LinkedIdentity[]
  workspaces: UserWorkspace[]
}

export interface LinkedIdentity {
  id: string
  provider: string
  providerEmail: string
  providerUserId: string
}

export interface UserWorkspace {
  id: string
  name: string
  role: string
}

export interface UpdatePlatformUserRequest {
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
