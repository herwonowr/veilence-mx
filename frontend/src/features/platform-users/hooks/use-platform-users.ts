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
  sort?: string
  order?: string
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
    }) => apiUpdatePlatformUser(id, req, confirmPassword),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: platformAdminKeys.users() })
      queryClient.invalidateQueries({
        queryKey: platformAdminKeys.user(variables.id),
      })
      toast.success("User updated")
    },
    onError: () => {
      toast.error("Failed to update user")
    },
  })
}
