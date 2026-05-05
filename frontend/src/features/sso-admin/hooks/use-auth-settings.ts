"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiGetAuthSettings,
  apiUpdateAuthSettings,
  platformAdminKeys,
} from "@/domains/platform-admin"
import type { UpdatePlatformAuthSettingsRequest } from "@/domains/platform-admin"
import { toast } from "sonner"

export const useAuthSettings = () => {
  const queryClient = useQueryClient()

  const { data: settingsRes, isLoading } = useQuery({
    queryKey: platformAdminKeys.authSettings(),
    queryFn: apiGetAuthSettings,
  })

  const updateMutation = useMutation({
    mutationFn: (req: UpdatePlatformAuthSettingsRequest) =>
      apiUpdateAuthSettings(req),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: platformAdminKeys.authSettings(),
      })
      toast.success("Auth settings updated")
    },
    onError: () => {
      toast.error("Failed to update auth settings")
    },
  })

  return {
    settings: settingsRes?.data ?? null,
    isLoading,
    updateSettings: updateMutation.mutate,
    isPending: updateMutation.isPending,
  }
}
