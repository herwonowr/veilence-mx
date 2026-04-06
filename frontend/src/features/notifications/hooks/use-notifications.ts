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
} from "@/lib/api-client"
import type { ApiResponse, Notification } from "@/types"

export const notificationKeys = {
  all: ["notifications"] as const,
  unreadCount: () => [...notificationKeys.all, "unread-count"] as const,
  list: (params?: Record<string, unknown>) =>
    [...notificationKeys.all, "list", params] as const,
}

export function useUnreadCount(
  enabled = true,
  options?: Partial<UseQueryOptions<ApiResponse<{ count: number }>>>
) {
  return useQuery({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => apiGetUnreadCount(),
    enabled,
    refetchInterval: 30_000, // poll every 30s
    staleTime: 10_000,
    ...options,
  })
}

export function useNotifications(
  params?: { page?: number; limit?: number },
  options?: Partial<UseQueryOptions<ApiResponse<Notification[]>>>
) {
  return useQuery({
    queryKey: notificationKeys.list(params as Record<string, unknown>),
    queryFn: () => apiListNotifications(params),
    ...options,
  })
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => apiMarkNotificationRead(id),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

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
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (notifications: Notification[]) => {
      const unread = notifications.filter((n) => !n.isRead)
      await Promise.allSettled(
        unread.map((n) => apiMarkNotificationRead(n.id))
      )
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: notificationKeys.all })

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
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}
