import { useMutation } from "@tanstack/react-query"
import {
  apiUpdateProfile,
  apiChangePassword,
  apiSendVerificationEmail,
} from "@/lib/api-client"
import { toast } from "sonner"

export function useUpdateProfile() {
  return useMutation({
    mutationFn: (data: { firstName: string; lastName: string }) =>
      apiUpdateProfile(data),
    onSuccess: () => {
      toast.success("Profile updated")
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to update profile")
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (data: { currentPassword: string; newPassword: string }) =>
      apiChangePassword(data),
    onSuccess: () => {
      toast.success("Password changed successfully")
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to change password")
    },
  })
}

export function useSendVerification() {
  return useMutation({
    mutationFn: () => apiSendVerificationEmail(),
    onSuccess: () => {
      toast.success("Verification email sent. Check your inbox.")
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to send verification email")
    },
  })
}
