export interface User {
  id: number
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
  id: number
  userId: number
  ipAddress: string
  userAgent: string
  createdAt: string
  lastActive: string
  expiresAt: string
  isCurrent: boolean
}

