import {
  useQuery,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { getPackages } from "@/domains/packages"
import type { ApiResponse } from "@/domains/common"
import type { Package } from "@/domains/packages"

export const adminPackageKeys = {
  all: ["admin-packages"] as const,
  list: (params?: Record<string, unknown>) =>
    [...adminPackageKeys.all, "list", params] as const,
}

export const useAdminPackages = (
  params?: {
    ecosystem?: string
    search?: string
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
  },
  options?: Partial<UseQueryOptions<ApiResponse<Package[]>>>
) =>
  useQuery({
    queryKey: adminPackageKeys.list(params as Record<string, unknown>),
    queryFn: () => getPackages(params),
    ...options,
  })
