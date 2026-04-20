export interface User {
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

export interface LoginResponse {
  user: User
  accessToken: string
  refreshToken: string
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

