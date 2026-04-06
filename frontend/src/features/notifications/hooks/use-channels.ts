import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiListChannels,
  apiCreateChannel,
  apiUpdateChannel,
  apiDeleteChannel,
  apiListRules,
  apiCreateRule,
  apiDeleteRule,
} from "@/lib/api-client"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/lib/utils"

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

export function useChannels(orgId: number | null) {
  return useQuery({
    queryKey: channelKeys.list(orgId ?? 0),
    queryFn: () => apiListChannels(orgId!),
    enabled: !!orgId,
    staleTime: 30_000,
  })
}

export function useCreateChannel(orgId: number) {
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

export function useUpdateChannel(orgId: number) {
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

export function useDeleteChannel(orgId: number) {
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

export function useRules(orgId: number | null) {
  return useQuery({
    queryKey: ruleKeys.list(orgId ?? 0),
    queryFn: () => apiListRules(orgId!),
    enabled: !!orgId,
    staleTime: 30_000,
  })
}

export function useCreateRule(orgId: number) {
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

export function useDeleteRule(orgId: number) {
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
