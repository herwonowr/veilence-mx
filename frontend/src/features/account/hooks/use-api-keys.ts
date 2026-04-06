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
} from "@/lib/api-client"
import type { ApiResponse, ApiKeyInfo, APIKeyScope } from "@/types"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/lib/utils"

export const apiKeyKeys = {
  all: ["api-keys"] as const,
  list: () => [...apiKeyKeys.all, "list"] as const,
}

export function useApiKeys(
  options?: Partial<UseQueryOptions<ApiResponse<ApiKeyInfo[]>>>
) {
  return useQuery({
    queryKey: apiKeyKeys.list(),
    queryFn: () => apiGetApiKeys(),
    ...options,
  })
}

export function useCreateApiKey() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { name: string; scope?: APIKeyScope; expiresAt?: string }) =>
      apiCreateApiKey(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list() })
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to create API key"))
    },
  })
}

export function useDeleteApiKey() {
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
