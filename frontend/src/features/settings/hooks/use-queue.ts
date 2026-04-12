import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  getQueueStats,
  getDeadJobs,
  retryDeadJobs,
  retryDeadJob,
} from "@/lib/api-client"
import type { ApiResponse, QueueStatsResponse, QueueJob } from "@/types"

export const queueKeys = {
  all: ["queue"] as const,
  stats: () => [...queueKeys.all, "stats"] as const,
  dead: (type?: string) => [...queueKeys.all, "dead", type] as const,
}

export function useQueueStats(
  options?: Partial<UseQueryOptions<ApiResponse<QueueStatsResponse>>>
) {
  return useQuery({
    queryKey: queueKeys.stats(),
    queryFn: () => getQueueStats(),
    ...options,
  })
}

export function useDeadJobs(
  type?: string,
  options?: Partial<UseQueryOptions<ApiResponse<QueueJob[]>>>
) {
  return useQuery({
    queryKey: queueKeys.dead(type),
    queryFn: () => getDeadJobs(type),
    ...options,
  })
}

export function useRetryDeadJobs() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (type?: string) => retryDeadJobs(type),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}

export function useRetryDeadJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (jobId: string) => retryDeadJob(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}
