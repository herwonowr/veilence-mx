import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { apiGetSessions, apiRevokeSession } from "@/lib/api-client"
import type { ApiResponse, Session } from "@/types"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/lib/error-sanitizer"

export const sessionKeys = {
  all: ["sessions"] as const,
  list: () => [...sessionKeys.all, "list"] as const,
}

export function useSessions(
  options?: Partial<UseQueryOptions<ApiResponse<Session[]>>>
) {
  return useQuery({
    queryKey: sessionKeys.list(),
    queryFn: () => apiGetSessions(),
    ...options,
  })
}

export function useRevokeSession(options?: {
  onRevoked?: () => void | Promise<void>
}) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiRevokeSession(id),
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: sessionKeys.list() })
      if (options?.onRevoked) {
        await options.onRevoked()
      } else {
        toast.success("Session revoked")
      }
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to revoke session"))
    },
  })
}
