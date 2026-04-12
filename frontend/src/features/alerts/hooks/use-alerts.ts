import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getAlerts, getAlert, updateAlertStatus, getAlertNotes, createAlertNote } from "@/domains/alerts"
import type { ApiResponse } from "@/domains/common"
import type { Alert, AlertNote } from "@/domains/alerts"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const alertKeys = {
  all: ["alerts"] as const,
  lists: () => [...alertKeys.all, "list"] as const,
  list: (params?: Record<string, unknown>) =>
    [...alertKeys.lists(), params] as const,
  detail: (id: number) => [...alertKeys.all, "detail", id] as const,
  notes: (alertId: number) => [...alertKeys.all, "notes", alertId] as const,
}

export const useAlert = (
  id: number,
  options?: Partial<UseQueryOptions<ApiResponse<Alert>>>
) =>
  useQuery({
    queryKey: alertKeys.detail(id),
    queryFn: () => getAlert(id),
    enabled: id > 0,
    ...options,
  })

export const useAlerts = (
  params?: {
    severity?: string
    status?: string
    search?: string
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
  },
  options?: Partial<UseQueryOptions<ApiResponse<Alert[]>>>
) =>
  useQuery({
    queryKey: alertKeys.list(params as Record<string, unknown>),
    queryFn: () => getAlerts(params),
    ...options,
  })

export const useUpdateAlert = () => {
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
      toast.error(sanitizeErrorMessage(error, "Failed to update alert status"))
    },
    onSuccess: (_data, variables) => {
      toast.success(`Alert ${variables.status}`)
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.lists() })
    },
  })
}

// ─── Alert Notes ──────────────────────────────────────────────

export const useAlertNotes = (
  alertId: number,
  options?: Partial<UseQueryOptions<ApiResponse<AlertNote[]>>>
) =>
  useQuery({
    queryKey: alertKeys.notes(alertId),
    queryFn: () => getAlertNotes(alertId),
    enabled: alertId > 0,
    ...options,
  })

export const useCreateAlertNote = (alertId: number) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (content: string) => createAlertNote(alertId, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.notes(alertId) })
      toast.success("Note added")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to add note"))
    },
  })
}
