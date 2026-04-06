import {
  useQuery,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getRelease } from "@/lib/api-client"
import type { ApiResponse, ReleaseDetail } from "@/types"

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
