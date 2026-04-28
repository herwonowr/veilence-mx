export interface SetupRequest {
  email: string
  password: string
  firstName: string
  lastName: string
  workspaceName: string
  workspaceSlug: string
}

export interface SetupResponse {
  user: {
    id: string
    email: string
    firstName: string
    lastName: string
    isActive: boolean
    emailVerified: boolean
    lastLoginAt: string | null
    createdAt: string
    updatedAt: string
  }
  accessToken: string
  refreshToken: string
  workspace: {
    id: string
    name: string
    slug: string
  }
}
