import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getAlerts, updateAlertStatus } from "@/lib/api-client"
import type { ApiResponse, Alert } from "@/types"
import { toast } from "sonner"

export const alertKeys = {
  all: ["alerts"] as const,
  lists: () => [...alertKeys.all, "list"] as const,
  list: (params?: Record<string, unknown>) =>
    [...alertKeys.lists(), params] as const,
}

export function useAlerts(
  params?: {
    severity?: string
    status?: string
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
  },
  options?: Partial<UseQueryOptions<ApiResponse<Alert[]>>>
) {
  return useQuery({
    queryKey: alertKeys.list(params as Record<string, unknown>),
    queryFn: () => getAlerts(params),
    ...options,
  })
}

export function useUpdateAlert() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      updateAlertStatus(id, status),
    onMutate: async ({ id, status }) => {
      // Cancel in-flight queries
      await queryClient.cancelQueries({ queryKey: alertKeys.lists() })

      // Snapshot all alert list queries for rollback
      const previousQueries = queryClient.getQueriesData<ApiResponse<Alert[]>>({
        queryKey: alertKeys.lists(),
      })

      // Optimistic update across all cached alert lists
      queryClient.setQueriesData<ApiResponse<Alert[]>>(
        { queryKey: alertKeys.lists() },
        (old) => {
          if (!old) return old
          return {
            ...old,
            data: old.data.map((alert) =>
              alert.id === id ? { ...alert, status: status as Alert["status"] } : alert
            ),
          }
        }
      )

      return { previousQueries }
    },
    onError: (error: Error, _variables, context) => {
      // Rollback on error
      if (context?.previousQueries) {
        for (const [queryKey, data] of context.previousQueries) {
          queryClient.setQueryData(queryKey, data)
        }
      }
      toast.error(error.message || "Failed to update alert status")
    },
    onSuccess: (_data, variables) => {
      toast.success(`Alert ${variables.status}`)
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.lists() })
    },
  })
}
