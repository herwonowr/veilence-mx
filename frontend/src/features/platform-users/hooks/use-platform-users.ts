"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiGetPlatformUsers,
  apiGetPlatformUser,
  apiUpdatePlatformUser,
  platformAdminKeys,
} from "@/domains/platform-admin"
import type { UpdatePlatformUserRequest } from "@/domains/platform-admin"
import { toast } from "sonner"

export const usePlatformUsers = (params: {
  page?: number
  pageSize?: number
  search?: string
  sortBy?: string
  sortDir?: string
  status?: string
  role?: string
}) =>
  useQuery({
    queryKey: platformAdminKeys.users(params),
    queryFn: () => apiGetPlatformUsers(params),
  })

export const usePlatformUser = (id: string) =>
  useQuery({
    queryKey: platformAdminKeys.user(id),
    queryFn: () => apiGetPlatformUser(id),
    enabled: !!id,
  })

export const useUpdatePlatformUser = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      req,
      confirmPassword,
    }: {
      id: string
      req: UpdatePlatformUserRequest
      confirmPassword: string
      successMessage?: string
    }) => apiUpdatePlatformUser(id, req, confirmPassword),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: platformAdminKeys.all })
      toast.success(variables.successMessage || "User updated")
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to update user")
    },
  })
}
