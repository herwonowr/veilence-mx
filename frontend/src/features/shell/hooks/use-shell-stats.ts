import {
  useQuery,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getDashboardStats } from "@/domains/dashboard"
import type { ApiResponse } from "@/domains/common"
import type { DashboardStats } from "@/domains/dashboard"

export const shellKeys = {
  all: ["shell"] as const,
  stats: () => [...shellKeys.all, "stats"] as const,
}

export const useShellDashboardStats = (
  options?: Partial<UseQueryOptions<ApiResponse<DashboardStats>>>
) =>
  useQuery({
    queryKey: shellKeys.stats(),
    queryFn: () => getDashboardStats(),
    staleTime: 30 * 1000,
    ...options,
  })
