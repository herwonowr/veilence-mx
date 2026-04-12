import { useMutation } from "@tanstack/react-query"
import {
  apiUpdateProfile,
  apiChangePassword,
  apiSendVerificationEmail,
} from "@/domains/auth"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const useUpdateProfile = () => {
  return useMutation({
    mutationFn: (data: { firstName: string; lastName: string }) =>
      apiUpdateProfile(data),
    onSuccess: () => {
      toast.success("Profile updated")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to update profile"))
    },
  })
}

export const useChangePassword = () => {
  return useMutation({
    mutationFn: (data: { currentPassword: string; newPassword: string }) =>
      apiChangePassword(data),
    onSuccess: () => {
      toast.success("Password changed successfully")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to change password"))
    },
  })
}

export const useSendVerification = () => {
  return useMutation({
    mutationFn: () => apiSendVerificationEmail(),
    onSuccess: () => {
      toast.success("Verification email sent. Check your inbox.")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to send verification email"))
    },
  })
}
