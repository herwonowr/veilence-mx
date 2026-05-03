export interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  isActive: boolean
  isSuperAdmin: boolean
  emailVerified: boolean
  mustChangePassword: boolean
  lastLoginAt: string | null
  createdAt: string
  updatedAt: string
}

export interface MeResponse extends User {
  allowedEmailDomains: string[]
}

export interface LoginResponse {
  user: User
  accessToken: string
  refreshToken: string
  mustChangePassword?: boolean
}

export interface ProfileUpdateRequest {
  firstName: string
  lastName: string
}

export interface PasswordChangeRequest {
  currentPassword: string
  newPassword: string
}

export interface Session {
  id: string
  userId: string
  ipAddress: string
  userAgent: string
  createdAt: string
  lastActive: string
  expiresAt: string
  isCurrent: boolean
}

