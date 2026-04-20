"use client"

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { apiGetSessions, apiRevokeSession } from "@/domains/auth"
import type { ApiResponse } from "@/domains/common"
import type { Session } from "@/domains/auth"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const sessionKeys = {
  all: ["sessions"] as const,
  list: () => [...sessionKeys.all, "list"] as const,
}

export const useSessions = (
  options?: Partial<UseQueryOptions<ApiResponse<Session[]>>>
) => {
  return useQuery({
    queryKey: sessionKeys.list(),
    queryFn: () => apiGetSessions(),
    ...options,
  })
}

export const useRevokeSession = (options?: {
  onRevoked?: () => void | Promise<void>
}) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiRevokeSession(id),
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
