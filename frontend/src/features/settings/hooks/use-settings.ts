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
import type { ApiResponse } from "@/domains/common"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const settingsKeys = {
  all: ["settings"] as const,
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
