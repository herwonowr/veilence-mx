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
} from "@/domains/queue"
import type { ApiResponse } from "@/domains/common"
import type { QueueStatsResponse, QueueJob } from "@/domains/queue"

export const queueKeys = {
  all: ["queue"] as const,
  stats: () => [...queueKeys.all, "stats"] as const,
  dead: (type?: string) => [...queueKeys.all, "dead", type] as const,
}

export const useQueueStats = (
  options?: Partial<UseQueryOptions<ApiResponse<QueueStatsResponse>>>
) => {
  return useQuery({
    queryKey: queueKeys.stats(),
    queryFn: () => getQueueStats(),
    ...options,
  })
}

export const useDeadJobs = (
  type?: string,
  options?: Partial<UseQueryOptions<ApiResponse<QueueJob[]>>>
) => {
  return useQuery({
    queryKey: queueKeys.dead(type),
    queryFn: () => getDeadJobs(type),
    ...options,
  })
}

export const useRetryDeadJobs = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (type?: string) => retryDeadJobs(type),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}

export const useRetryDeadJob = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (jobId: string) => retryDeadJob(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}
