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
} from "@/lib/api-client"
import type { ApiResponse } from "@/types"
import { toast } from "sonner"

export const settingsKeys = {
  all: ["settings"] as const,
}

export function useSettings(
  options?: Partial<UseQueryOptions<ApiResponse<Record<string, string>>>>
) {
  return useQuery({
    queryKey: settingsKeys.all,
    queryFn: () => getSettings(),
    ...options,
  })
}

export function useUpdateSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (settings: Record<string, string>) => updateSettings(settings),
    onSuccess: (response) => {
      queryClient.setQueryData(settingsKeys.all, response)
      toast.success("Settings saved")
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to save settings")
    },
  })
}

export function useReanalyzeAll() {
  return useMutation({
    mutationFn: () => reanalyzeAll(),
  })
}
