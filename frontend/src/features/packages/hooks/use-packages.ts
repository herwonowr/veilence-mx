"use client"

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
  blockPackage,
  unblockPackage,
  getPackageReleases,
  discoverPackages,
  bulkImportPackages,
  getAnalysisHistory,
  getPackageSuggestions,
  approvePackage,
  rejectPackage,
  bulkApprovePackages,
  getStalePackages,
} from "@/domains/packages"
import type { ApiResponse } from "@/domains/common"
import type { Package, Release, AnalysisHistoryEntry, StalePackage } from "@/domains/packages"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const packageKeys = {
  all: ["packages"] as const,
  lists: () => [...packageKeys.all, "list"] as const,
  list: (params?: Record<string, unknown>) =>
    [...packageKeys.lists(), params] as const,
  details: () => [...packageKeys.all, "detail"] as const,
  detail: (id: string) => [...packageKeys.details(), id] as const,
  releases: (packageId: string, page?: number, limit?: number) =>
    [...packageKeys.all, "releases", packageId, page, limit] as const,
  analysisHistory: (packageId: string) =>
    [...packageKeys.all, "analysis-history", packageId] as const,
  suggestions: (params?: Record<string, unknown>) =>
    [...packageKeys.all, "suggestions", params] as const,
  stale: (months?: number) =>
    [...packageKeys.all, "stale", months] as const,
  suggestionCount: () =>
    [...packageKeys.all, "suggestion-count"] as const,
}

export const usePackages = (
  params?: {
    ecosystem?: string
    search?: string
    status?: string
    source?: string
    page?: number
    limit?: number
    sortBy?: string
    sortDir?: string
  },
  options?: Partial<UseQueryOptions<ApiResponse<Package[]>>>
) =>
  useQuery({
    queryKey: packageKeys.list(params as Record<string, unknown>),
    queryFn: () => getPackages(params),
    ...options,
  })

export const usePackage = (
  id: string,
  options?: Partial<UseQueryOptions<ApiResponse<Package>>>
) =>
  useQuery({
    queryKey: packageKeys.detail(id),
    queryFn: () => getPackage(id),
    enabled: !!id,
    ...options,
  })

export const usePackageReleases = (
  packageId: string,
  page = 1,
  limit = 50,
  options?: Partial<UseQueryOptions<ApiResponse<Release[]>>>
) =>
  useQuery({
    queryKey: packageKeys.releases(packageId, page, limit),
    queryFn: () => getPackageReleases(packageId, page, limit),
    enabled: !!packageId,
    ...options,
  })

export const useCreatePackage = () => {
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

export const useDeletePackage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deletePackage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Package removed from monitoring")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to remove package"))
    },
  })
}

export const useBlockPackage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      blockPackage(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Package blocked")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to block package"))
    },
  })
}

export const useUnblockPackage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => unblockPackage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Package unblocked")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to unblock package"))
    },
  })
}

export const useDiscoverPackages = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ecosystem?: string) => discoverPackages(ecosystem),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.lists() })
      toast.success("Discovery started")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to trigger discovery"))
    },
  })
}

export const useBulkImportPackages = () => {
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

export const useAnalysisHistory = (
  packageId: string,
  options?: Partial<UseQueryOptions<ApiResponse<AnalysisHistoryEntry[]>>>
) =>
  useQuery({
    queryKey: packageKeys.analysisHistory(packageId),
    queryFn: () => getAnalysisHistory(packageId),
    enabled: !!packageId,
    ...options,
  })

export const usePackageSuggestions = (
  params?: {
    page?: number
    limit?: number
    search?: string
    ecosystem?: string
    sortBy?: string
    sortDir?: string
  },
  options?: Partial<UseQueryOptions<ApiResponse<Package[]>>>
) =>
  useQuery({
    queryKey: packageKeys.suggestions(params as Record<string, unknown>),
    queryFn: () => getPackageSuggestions(params),
    ...options,
  })

export const useApprovePackage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => approvePackage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.all })
      toast.success("Package approved and added to monitoring")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to approve package"))
    },
  })
}

export const useRejectPackage = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => rejectPackage(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: packageKeys.all })
      toast.success("Suggestion rejected")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to reject suggestion"))
    },
  })
}

export const useBulkApprovePackages = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (params: { packageIds: string[] } | { ecosystem: string } | { approveAll: true }) =>
      bulkApprovePackages(params),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: packageKeys.all })
      toast.success(`Approved ${result.data.approved} package(s)`)
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to bulk approve packages"))
    },
  })
}

export const useStalePackages = (
  months = 6,
  options?: Partial<UseQueryOptions<ApiResponse<StalePackage[]>>>
) =>
  useQuery({
    queryKey: packageKeys.stale(months),
    queryFn: () => getStalePackages(months),
    ...options,
  })

export const useSuggestionCount = () =>
  useQuery({
    queryKey: packageKeys.suggestionCount(),
    queryFn: async () => {
      const res = await getPackageSuggestions({ limit: 1 })
      return res.meta?.total ?? 0
    },
    staleTime: 30_000,
  })
