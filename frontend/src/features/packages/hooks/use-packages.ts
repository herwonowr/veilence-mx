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
  bulkImportPackages,
  getAnalysisHistory,
} from "@/lib/api-client"
import type { ApiResponse, Package, Release, AnalysisHistoryEntry } from "@/types"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/lib/error-sanitizer"

export const packageKeys = {
  all: ["packages"] as const,
  lists: () => [...packageKeys.all, "list"] as const,
  list: (params?: Record<string, unknown>) =>
    [...packageKeys.lists(), params] as const,
  details: () => [...packageKeys.all, "detail"] as const,
  detail: (id: number) => [...packageKeys.details(), id] as const,
  releases: (packageId: number, page?: number, limit?: number) =>
    [...packageKeys.all, "releases", packageId, page, limit] as const,
  analysisHistory: (packageId: number) =>
    [...packageKeys.all, "analysis-history", packageId] as const,
}

export function usePackages(
  params?: {
    ecosystem?: string
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
    mutationFn: ({ name, ecosystem }: { name: string; ecosystem: string }) =>
      createPackage(name, ecosystem),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Package added")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to add package"))
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
      toast.error(sanitizeErrorMessage(error, "Failed to delete package"))
    },
  })
}

export function useSyncTopPackages() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ecosystem?: string) => syncTopPackages(ecosystem),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Top packages synced")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to sync packages"))
    },
  })
}

export function useBulkImportPackages() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      format,
      content,
    }: {
      format: "requirements_txt" | "package_json" | "list"
      content: string
    }) => bulkImportPackages(format, content),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      const { imported, skipped } = result.data
      toast.success(`Imported ${imported} package(s)${skipped > 0 ? `, ${skipped} skipped` : ""}`)
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to import packages"))
    },
  })
}

export function useAnalysisHistory(
  packageId: number,
  options?: Partial<UseQueryOptions<ApiResponse<AnalysisHistoryEntry[]>>>
) {
  return useQuery({
    queryKey: packageKeys.analysisHistory(packageId),
    queryFn: () => getAnalysisHistory(packageId),
    enabled: packageId > 0,
    ...options,
  })
}
