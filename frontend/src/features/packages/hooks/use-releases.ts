import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getRelease, reanalyzeRelease } from "@/domains/releases"
import type { ApiResponse } from "@/domains/common"
import type { ReleaseDetail } from "@/domains/releases"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const releaseKeys = {
  all: ["releases"] as const,
  detail: (id: number) => [...releaseKeys.all, "detail", id] as const,
}

export const useRelease = (
  id: number,
  options?: Partial<UseQueryOptions<ApiResponse<ReleaseDetail>>>
) =>
  useQuery({
    queryKey: releaseKeys.detail(id),
    queryFn: () => getRelease(id),
    enabled: id > 0,
    ...options,
  })

export const useReanalyzeRelease = (releaseId: number) => {
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
