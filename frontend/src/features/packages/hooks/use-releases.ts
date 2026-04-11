import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getRelease, reanalyzeRelease } from "@/lib/api-client"
import type { ApiResponse, ReleaseDetail } from "@/types"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/lib/error-sanitizer"

export const releaseKeys = {
  all: ["releases"] as const,
  detail: (id: number) => [...releaseKeys.all, "detail", id] as const,
}

export function useRelease(
  id: number,
  options?: Partial<UseQueryOptions<ApiResponse<ReleaseDetail>>>
) {
  return useQuery({
    queryKey: releaseKeys.detail(id),
    queryFn: () => getRelease(id),
    enabled: id > 0,
    ...options,
  })
}

export function useReanalyzeRelease(releaseId: number) {
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
