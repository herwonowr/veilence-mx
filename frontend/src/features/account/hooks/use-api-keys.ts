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
import type { ApiResponse, ApiKeyInfo } from "@/types"
import { toast } from "sonner"

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
    mutationFn: (data: { name: string; expiresAt?: string }) =>
      apiCreateApiKey(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list() })
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create API key")
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
      toast.error(error.message || "Failed to delete API key")
    },
  })
}
