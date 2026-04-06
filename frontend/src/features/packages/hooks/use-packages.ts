import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  getPackages,
  getPackage,
  createPackage,
  deletePackage,
  getPackageReleases,
  syncTopPackages,
} from "@/lib/api-client"
import type { ApiResponse, Package, Release } from "@/types"
import { toast } from "sonner"

export const packageKeys = {
  all: ["packages"] as const,
  lists: () => [...packageKeys.all, "list"] as const,
  list: (params?: Record<string, unknown>) =>
    [...packageKeys.lists(), params] as const,
  details: () => [...packageKeys.all, "detail"] as const,
  detail: (id: number) => [...packageKeys.details(), id] as const,
  releases: (packageId: number, page?: number, limit?: number) =>
    [...packageKeys.all, "releases", packageId, page, limit] as const,
}

export function usePackages(
  params?: {
    registry?: string
    search?: string
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
  },
  options?: Partial<UseQueryOptions<ApiResponse<Package[]>>>
) {
  return useQuery({
    queryKey: packageKeys.list(params as Record<string, unknown>),
    queryFn: () => getPackages(params),
    ...options,
  })
}

export function usePackage(
  id: number,
  options?: Partial<UseQueryOptions<ApiResponse<Package>>>
) {
  return useQuery({
    queryKey: packageKeys.detail(id),
    queryFn: () => getPackage(id),
    enabled: id > 0,
    ...options,
  })
}

export function usePackageReleases(
  packageId: number,
  page = 1,
  limit = 50,
  options?: Partial<UseQueryOptions<ApiResponse<Release[]>>>
) {
  return useQuery({
    queryKey: packageKeys.releases(packageId, page, limit),
    queryFn: () => getPackageReleases(packageId, page, limit),
    enabled: packageId > 0,
    ...options,
  })
}

export function useCreatePackage() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ name, registry }: { name: string; registry: string }) =>
      createPackage(name, registry),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Package added")
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to add package")
    },
  })
}

export function useDeletePackage() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => deletePackage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Package deleted")
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to delete package")
    },
  })
}

export function useSyncTopPackages() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (registry?: string) => syncTopPackages(registry),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Top packages synced")
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to sync packages")
    },
  })
}
