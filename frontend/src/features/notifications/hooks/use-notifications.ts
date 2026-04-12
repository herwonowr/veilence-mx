"use client"

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import {
  apiGetUnreadCount,
  apiListNotifications,
  apiMarkNotificationRead,
  apiMarkAllNotificationsRead,
} from "@/domains/notifications"
import type { ApiResponse } from "@/domains/common"
import type { Notification } from "@/domains/notifications"

export const notificationKeys = {
  all: ["notifications"] as const,
  unreadCount: () => [...notificationKeys.all, "unread-count"] as const,
  list: (params?: Record<string, unknown>) =>
    [...notificationKeys.all, "list", params] as const,
}

export const useUnreadCount = (
  enabled = true,
  options?: Partial<UseQueryOptions<ApiResponse<{ count: number }>>>
) => {
  return useQuery({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => apiGetUnreadCount(),
    enabled,
    refetchInterval: 30_000, // poll every 30s
    staleTime: 10_000,
    ...options,
  })
}

export const useNotifications = (
  params?: { page?: number; limit?: number },
  options?: Partial<UseQueryOptions<ApiResponse<Notification[]>>>
) => {
  return useQuery({
    queryKey: notificationKeys.list(params as Record<string, unknown>),
    queryFn: () => apiListNotifications(params),
    ...options,
  })
}

export const useMarkNotificationRead = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiMarkNotificationRead(id),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

      // Snapshot for rollback
      const previousNotifications = queryClient.getQueriesData<ApiResponse<Notification[]>>({
        queryKey: notificationKeys.all,
      })
      const previousUnreadCount = queryClient.getQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount()
      )

      // Optimistic update: mark as read in cached list
      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.all },
        (old) => {
          if (!old?.data) return old
          return {
            ...old,
            data: old.data.map((n) =>
              n.id === id ? { ...n, isRead: true } : n
            ),
          }
        }
      )

      // Optimistic update: decrement unread count
      queryClient.setQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount(),
        (old) => {
          if (!old?.data) return old
          return { ...old, data: { count: Math.max(0, old.data.count - 1) } }
        }
      )

      return { previousNotifications, previousUnreadCount }
    },
    onError: (_error, _id, context) => {
      // Rollback on error
      if (context?.previousNotifications) {
        for (const [queryKey, data] of context.previousNotifications) {
          queryClient.setQueryData(queryKey, data)
        }
      }
      if (context?.previousUnreadCount) {
        queryClient.setQueryData(notificationKeys.unreadCount(), context.previousUnreadCount)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export const useMarkAllNotificationsRead = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      await apiMarkAllNotificationsRead()
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

      // Snapshot for rollback
      const previousNotifications = queryClient.getQueriesData<ApiResponse<Notification[]>>({
        queryKey: notificationKeys.all,
      })
      const previousUnreadCount = queryClient.getQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount()
      )

      // Optimistic: mark everything read
      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.all },
        (old) => {
          if (!old?.data) return old
          return {
            ...old,
            data: old.data.map((n) => ({ ...n, isRead: true })),
          }
        }
      )

      queryClient.setQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount(),
        (old) => {
          if (!old?.data) return old
          return { ...old, data: { count: 0 } }
        }
      )

      return { previousNotifications, previousUnreadCount }
    },
    onError: (_error, _vars, context) => {
      // Rollback on error
      if (context?.previousNotifications) {
        for (const [queryKey, data] of context.previousNotifications) {
          queryClient.setQueryData(queryKey, data)
        }
      }
      if (context?.previousUnreadCount) {
        queryClient.setQueryData(notificationKeys.unreadCount(), context.previousUnreadCount)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}
