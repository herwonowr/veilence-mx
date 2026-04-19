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
  apiDeleteNotification,
  apiDeleteAllNotifications,
  apiDeleteBatchNotifications,
} from "@/domains/notifications"
import type { ApiResponse } from "@/domains/common"
import type { Notification } from "@/domains/notifications"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

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
  params?: { page?: number; limit?: number; unread?: boolean },
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
      // Use notificationKeys.list() (not .all) to only match list queries -
      // .all also matches the unread-count query whose data is not an array.
      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.list() },
        (old) => {
          if (!old?.data || !Array.isArray(old.data)) return old
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
      toast.error(sanitizeErrorMessage(_error, "Failed to mark notification as read"))
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
      // Use notificationKeys.list() (not .all) to only match list queries -
      // .all also matches the unread-count query whose data is not an array.
      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.list() },
        (old) => {
          if (!old?.data || !Array.isArray(old.data)) return old
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
      toast.error(sanitizeErrorMessage(_error, "Failed to mark all notifications as read"))
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export const useDeleteNotification = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiDeleteNotification(id),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

      const previousNotifications = queryClient.getQueriesData<ApiResponse<Notification[]>>({
        queryKey: notificationKeys.list(),
      })
      const previousUnreadCount = queryClient.getQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount()
      )

      let wasUnread = false
      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.list() },
        (old) => {
          if (!old?.data || !Array.isArray(old.data)) return old
          const target = old.data.find((n) => n.id === id)
          if (target && !target.isRead) wasUnread = true
          return {
            ...old,
            data: old.data.filter((n) => n.id !== id),
          }
        }
      )

      if (wasUnread) {
        queryClient.setQueryData<ApiResponse<{ count: number }>>(
          notificationKeys.unreadCount(),
          (old) => {
            if (!old?.data) return old
            return { ...old, data: { count: Math.max(0, old.data.count - 1) } }
          }
        )
      }

      return { previousNotifications, previousUnreadCount }
    },
    onError: (_error, _id, context) => {
      if (context?.previousNotifications) {
        for (const [queryKey, data] of context.previousNotifications) {
          queryClient.setQueryData(queryKey, data)
        }
      }
      if (context?.previousUnreadCount) {
        queryClient.setQueryData(notificationKeys.unreadCount(), context.previousUnreadCount)
      }
      toast.error(sanitizeErrorMessage(_error, "Failed to delete notification"))
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export const useDeleteAllNotifications = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async () => {
      await apiDeleteAllNotifications()
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

      const previousNotifications = queryClient.getQueriesData<ApiResponse<Notification[]>>({
        queryKey: notificationKeys.list(),
      })
      const previousUnreadCount = queryClient.getQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount()
      )

      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.list() },
        (old) => {
          if (!old?.data || !Array.isArray(old.data)) return old
          return { ...old, data: [] }
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
      if (context?.previousNotifications) {
        for (const [queryKey, data] of context.previousNotifications) {
          queryClient.setQueryData(queryKey, data)
        }
      }
      if (context?.previousUnreadCount) {
        queryClient.setQueryData(notificationKeys.unreadCount(), context.previousUnreadCount)
      }
      toast.error(sanitizeErrorMessage(_error, "Failed to delete all notifications"))
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export const useDeleteBatchNotifications = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ids: number[]) => apiDeleteBatchNotifications(ids),
    onMutate: async (ids) => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

      const previousNotifications = queryClient.getQueriesData<ApiResponse<Notification[]>>({
        queryKey: notificationKeys.list(),
      })
      const previousUnreadCount = queryClient.getQueryData<ApiResponse<{ count: number }>>(
        notificationKeys.unreadCount()
      )

      const idsSet = new Set(ids)
      let unreadRemoved = 0

      queryClient.setQueriesData<ApiResponse<Notification[]>>(
        { queryKey: notificationKeys.list() },
        (old) => {
          if (!old?.data || !Array.isArray(old.data)) return old
          for (const n of old.data) {
            if (idsSet.has(n.id) && !n.isRead) unreadRemoved++
          }
          return {
            ...old,
            data: old.data.filter((n) => !idsSet.has(n.id)),
          }
        }
      )

      if (unreadRemoved > 0) {
        queryClient.setQueryData<ApiResponse<{ count: number }>>(
          notificationKeys.unreadCount(),
          (old) => {
            if (!old?.data) return old
            return { ...old, data: { count: Math.max(0, old.data.count - unreadRemoved) } }
          }
        )
      }

      return { previousNotifications, previousUnreadCount }
    },
    onError: (_error, _ids, context) => {
      if (context?.previousNotifications) {
        for (const [queryKey, data] of context.previousNotifications) {
          queryClient.setQueryData(queryKey, data)
        }
      }
      if (context?.previousUnreadCount) {
        queryClient.setQueryData(notificationKeys.unreadCount(), context.previousUnreadCount)
      }
      toast.error(sanitizeErrorMessage(_error, "Failed to delete notifications"))
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}
