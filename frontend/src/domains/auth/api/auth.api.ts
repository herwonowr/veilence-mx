import { fetchApi, getStoredRefreshToken, type ApiResponse } from "@/core"
import type { User, LoginResponse, ProfileUpdateRequest, PasswordChangeRequest, Session } from "@/domains/auth/types/auth.types"

export const apiRefreshToken = async (
  refreshToken: string
): Promise<ApiResponse<LoginResponse>> =>
  fetchApi<LoginResponse>("/api/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refreshToken }),
    skipAuth: true,
  })

export const apiUpdateProfile = async (
  data: ProfileUpdateRequest
): Promise<ApiResponse<User>> =>
  fetchApi<User>("/api/auth/me", {
    method: "PUT",
    body: JSON.stringify(data),
  })

export const apiChangePassword = async (
  data: PasswordChangeRequest
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>("/api/auth/change-password", {
    method: "POST",
    body: JSON.stringify(data),
  })

export const apiSendVerificationEmail = async (): Promise<
  ApiResponse<{ message: string }>
> =>
  fetchApi<{ message: string }>("/api/auth/send-verification", {
    method: "POST",
  })

export const apiSendVerificationEmailByEmail = async (
  email: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>("/api/auth/resend-verification", {
    method: "POST",
    body: JSON.stringify({ email }),
    skipAuth: true,
  })

export const apiForgotPassword = async (
  email: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>("/api/auth/forgot-password", {
    method: "POST",
    body: JSON.stringify({ email }),
    skipAuth: true,
  })

export const apiResetPassword = async (
  token: string,
  password: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>("/api/auth/reset-password", {
    method: "POST",
    body: JSON.stringify({ token, newPassword: password }),
    skipAuth: true,
  })

export const apiGetSessions = async (): Promise<ApiResponse<Session[]>> => {
  const headers: Record<string, string> = {}
  const refreshToken = getStoredRefreshToken()
  if (refreshToken) {
    headers["X-Refresh-Token"] = refreshToken
  }
  return fetchApi<Session[]>("/api/auth/sessions", { headers })
}

export const apiRevokeSession = async (
  id: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>(`/api/auth/sessions/${id}`, {
    method: "DELETE",
  })

export const apiVerifyEmail = async (
  token: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>("/api/auth/verify-email", {
    method: "POST",
    body: JSON.stringify({ token }),
    skipAuth: true,
  })
