"use client"

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  apiGetApiKeys,
  apiCreateApiKey,
  apiDeleteApiKey,
  apiGetCurrentMemberRole,
} from "@/domains/account"
import type { ApiResponse } from "@/domains/common"
import type { ApiKeyInfo, APIKeyRole } from "@/domains/account"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"
import { useAuth } from "@/core/providers/auth-provider"

export const apiKeyKeys = {
  all: ["api-keys"] as const,
  list: () => [...apiKeyKeys.all, "list"] as const,
  currentRole: (workspaceId: number) => [...apiKeyKeys.all, "current-role", workspaceId] as const,
}

export const useApiKeys = (
  options?: Partial<UseQueryOptions<ApiResponse<ApiKeyInfo[]>>>
) => {
  return useQuery({
    queryKey: apiKeyKeys.list(),
    queryFn: () => apiGetApiKeys(),
    ...options,
  })
}

export const useCurrentWorkspaceRole = (): APIKeyRole | null => {
  const { currentWorkspace } = useAuth()

  const { data } = useQuery({
    queryKey: apiKeyKeys.currentRole(currentWorkspace?.id ?? 0),
    queryFn: () => apiGetCurrentMemberRole(currentWorkspace!.id),
    enabled: !!currentWorkspace,
  })

  const roleName = data?.data?.role?.toLowerCase()
  if (roleName === "owner" || roleName === "admin" || roleName === "member" || roleName === "viewer") {
    return roleName
  }
  return null
}

export const useCreateApiKey = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { name: string; role?: APIKeyRole; expiresAt?: string }) =>
      apiCreateApiKey(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list() })
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to create API key"))
    },
  })
}

export const useDeleteApiKey = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiDeleteApiKey(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list() })
      toast.success("API key deleted")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to delete API key"))
    },
  })
}
