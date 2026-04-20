"use client"

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
  list: (workspaceId: string) => [...channelKeys.lists(), workspaceId] as const,
}

export const ruleKeys = {
  all: ["rules"] as const,
  lists: () => [...ruleKeys.all, "list"] as const,
  list: (workspaceId: string) => [...ruleKeys.lists(), workspaceId] as const,
}

export const useChannels = (workspaceId: string | null) => {
  return useQuery({
    queryKey: channelKeys.list(workspaceId ?? ""),
    queryFn: () => apiListChannels(workspaceId!),
    enabled: !!workspaceId,
    staleTime: 30_000,
  })
}

export const useCreateChannel = (workspaceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { name: string; type: string; config: string }) =>
      apiCreateChannel(workspaceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) })
      toast.success("Channel created")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to create channel"))
    },
  })
}

export const useUpdateChannel = (workspaceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...data
    }: {
      id: string
      name: string
      config: string
      isActive: boolean
    }) => apiUpdateChannel(workspaceId, id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) })
      toast.success("Channel updated")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to update channel"))
    },
  })
}

export const useDeleteChannel = (workspaceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiDeleteChannel(workspaceId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) })
      toast.success("Channel deleted")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to delete channel"))
    },
  })
}

export const useRules = (workspaceId: string | null) => {
  return useQuery({
    queryKey: ruleKeys.list(workspaceId ?? ""),
    queryFn: () => apiListRules(workspaceId!),
    enabled: !!workspaceId,
    staleTime: 30_000,
  })
}

export const useCreateRule = (workspaceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { channelId: string; severity: string }) =>
      apiCreateRule(workspaceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ruleKeys.list(workspaceId) })
      toast.success("Rule created")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to create rule"))
    },
  })
}

export const useDeleteRule = (workspaceId: string) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiDeleteRule(workspaceId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ruleKeys.list(workspaceId) })
      toast.success("Rule deleted")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to delete rule"))
    },
  })
}

export const useTestChannel = (workspaceId: string) => {
  return useMutation({
    mutationFn: (channelId: string) => testNotificationChannel(workspaceId, channelId),
    onSuccess: () => {
      toast.success("Test notification sent")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to send test notification"))
    },
  })
}
