"use client"

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getRecentReleases } from "@/domains/dashboard"
import { getRelease, reanalyzeRelease } from "@/domains/releases"
import type { ApiResponse } from "@/domains/common"
import type { RecentRelease } from "@/domains/dashboard"
import type { ReleaseDetail } from "@/domains/releases"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const releaseKeys = {
  all: ["releases"] as const,
  list: (params?: Record<string, unknown>) =>
    [...releaseKeys.all, "list", params] as const,
  detail: (id: string) => [...releaseKeys.all, "detail", id] as const,
}

export const useRecentReleases = (
  params?: {
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
    search?: string
    ecosystem?: string
    status?: string
    classification?: string
    latestPerPackage?: boolean
  },
  options?: Partial<UseQueryOptions<ApiResponse<RecentRelease[]>>>
) =>
  useQuery({
    queryKey: releaseKeys.list(params as Record<string, unknown>),
    queryFn: () => getRecentReleases(params),
    staleTime: 30 * 1000,
    ...options,
  })

export const useRelease = (
  id: string,
  options?: Partial<UseQueryOptions<ApiResponse<ReleaseDetail>>>
) =>
  useQuery({
    queryKey: releaseKeys.detail(id),
    queryFn: () => getRelease(id),
    enabled: !!id,
    ...options,
  })

export const useReanalyzeRelease = (releaseId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => reanalyzeRelease(releaseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: releaseKeys.detail(releaseId) })
      toast.success("Re-analysis queued")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to queue re-analysis"))
    },
  })
}
