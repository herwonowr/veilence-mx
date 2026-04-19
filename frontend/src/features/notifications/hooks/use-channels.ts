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
  list: (workspaceId: number) => [...channelKeys.lists(), workspaceId] as const,
}

export const ruleKeys = {
  all: ["rules"] as const,
  lists: () => [...ruleKeys.all, "list"] as const,
  list: (workspaceId: number) => [...ruleKeys.lists(), workspaceId] as const,
}

export const useChannels = (workspaceId: number | null) => {
  return useQuery({
    queryKey: channelKeys.list(workspaceId ?? 0),
    queryFn: () => apiListChannels(workspaceId!),
    enabled: !!workspaceId,
    staleTime: 30_000,
  })
}

export const useCreateChannel = (workspaceId: number) => {
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

export const useUpdateChannel = (workspaceId: number) => {
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

export const useDeleteChannel = (workspaceId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiDeleteChannel(workspaceId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: channelKeys.list(workspaceId) })
      toast.success("Channel deleted")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to delete channel"))
    },
  })
}

export const useRules = (workspaceId: number | null) => {
  return useQuery({
    queryKey: ruleKeys.list(workspaceId ?? 0),
    queryFn: () => apiListRules(workspaceId!),
    enabled: !!workspaceId,
    staleTime: 30_000,
  })
}

export const useCreateRule = (workspaceId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { channelId: number; severity: string }) =>
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

export const useDeleteRule = (workspaceId: number) => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => apiDeleteRule(workspaceId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ruleKeys.list(workspaceId) })
      toast.success("Rule deleted")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to delete rule"))
    },
  })
}

export const useTestChannel = (workspaceId: number) => {
  return useMutation({
    mutationFn: (channelId: number) => testNotificationChannel(workspaceId, channelId),
    onSuccess: () => {
      toast.success("Test notification sent")
    },
    onError: (err: Error) => {
      toast.error(sanitizeErrorMessage(err, "Failed to send test notification"))
    },
  })
}
