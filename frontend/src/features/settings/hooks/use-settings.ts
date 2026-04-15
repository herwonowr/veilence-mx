"use client"

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  getSettings,
  updateSettings,
  reanalyzeAll,
} from "@/domains/settings"
import { discoverPackages, getPackages, getPackageSuggestions } from "@/domains/packages"
import type { ApiResponse } from "@/domains/common"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const settingsKeys = {
  all: ["settings"] as const,
  packageSummary: () => [...settingsKeys.all, "package-summary"] as const,
}

export const useSettings = (
  options?: Partial<UseQueryOptions<ApiResponse<Record<string, string>>>>
) => {
  return useQuery({
    queryKey: settingsKeys.all,
    queryFn: () => getSettings(),
    ...options,
  })
}

export const useUpdateSettings = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (settings: Record<string, string>) => updateSettings(settings),
    onSuccess: (response) => {
      queryClient.setQueryData(settingsKeys.all, response)
      toast.success("Settings saved")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to save settings"))
    },
  })
}

export const useReanalyzeAll = () => {
  return useMutation({
    mutationFn: () => reanalyzeAll(),
  })
}

export const useDiscoverNow = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ecosystem?: string) => discoverPackages(ecosystem),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["packages"] })
      queryClient.invalidateQueries({ queryKey: settingsKeys.packageSummary() })
      toast.success("Discovery started")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to trigger discovery"))
    },
  })
}

export const usePackageCountSummary = () =>
  useQuery({
    queryKey: settingsKeys.packageSummary(),
    queryFn: async () => {
      const [activeRes, suggestionsRes] = await Promise.all([
        getPackages({ status: "active", limit: 1 }),
        getPackageSuggestions({ limit: 1 }),
      ])
      return {
        activeCount: activeRes.meta?.total ?? 0,
        suggestionsCount: suggestionsRes.meta?.total ?? 0,
      }
    },
    staleTime: 30 * 1000,
  })
