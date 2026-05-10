"use client"

import { useQuery } from "@tanstack/react-query"
import { getPipelineStatus } from "@/domains/releases"
import type { ApiResponse } from "@/domains/common"
import type { PipelineStatus } from "@/domains/releases"

export const pipelineStatusKeys = {
  all: ["pipeline-status"] as const,
}

export const usePipelineStatus = () => {
  const query = useQuery<ApiResponse<PipelineStatus>>({
    queryKey: pipelineStatusKeys.all,
    queryFn: () => getPipelineStatus(),
    refetchInterval: (query) => {
      const status = query.state.data?.data
      if (!status) return 10_000
      const hasActive = status.pending > 0 || status.diffing > 0 || status.analyzing > 0
      return hasActive ? 10_000 : false
    },
    staleTime: 5_000,
  })

  const status = query.data?.data
  const hasActiveJobs = !!(status && (status.pending > 0 || status.diffing > 0 || status.analyzing > 0))

  return {
    ...query,
    status,
    hasActiveJobs,
  }
}
