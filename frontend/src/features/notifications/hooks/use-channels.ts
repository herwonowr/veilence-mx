import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiListChannels,
  apiCreateChannel,
  apiUpdateChannel,
  apiDeleteChannel,
  apiListRules,
  apiCreateRule,
  apiDeleteRule,
  testNotificationChannel,
} from "@/domains/notifications"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const channelKeys = {
  all: ["channels"] as const,
  lists: () => [...channelKeys.all, "list"] as const,
  list: (orgId: number) => [...channelKeys.lists(), orgId] as const,
}

export const ruleKeys = {
  all: ["rules"] as const,
  lists: () => [...ruleKeys.all, "list"] as const,
  list: (orgId: number) => [...ruleKeys.lists(), orgId] as const,
}

export const useChannels = (orgId: number | null) => {
  return useQuery({
    queryKey: channelKeys.list(orgId ?? 0),
    queryFn: () => apiListChannels(orgId!),
    enabled: !!orgId,
    staleTime: 30_000,
  })
}

export const useCreateChannel = (orgId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { name: string; type: string; config: string }) =>
      apiCreateChannel(orgId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(orgId) })
      toast.success("Channel created")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to create channel"))
    },
  })
}

export const useUpdateChannel = (orgId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...data
    }: {
      id: number
      name: string
      config: string
      isActive: boolean
    }) => apiUpdateChannel(orgId, id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(orgId) })
      toast.success("Channel updated")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to update channel"))
    },
  })
}

export const useDeleteChannel = (orgId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiDeleteChannel(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(orgId) })
      toast.success("Channel deleted")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to delete channel"))
    },
  })
}

export const useRules = (orgId: number | null) => {
  return useQuery({
    queryKey: ruleKeys.list(orgId ?? 0),
    queryFn: () => apiListRules(orgId!),
    enabled: !!orgId,
    staleTime: 30_000,
  })
}

export const useCreateRule = (orgId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { channelId: number; severity: string }) =>
      apiCreateRule(orgId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ruleKeys.list(orgId) })
      toast.success("Rule created")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to create rule"))
    },
  })
}

export const useDeleteRule = (orgId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiDeleteRule(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ruleKeys.list(orgId) })
      toast.success("Rule deleted")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to delete rule"))
    },
  })
}

export const useTestChannel = (orgId: number) => {
  return useMutation({
    mutationFn: (channelId: number) => testNotificationChannel(orgId, channelId),
    onSuccess: () => {
      toast.success("Test notification sent")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to send test notification"))
    },
  })
}
