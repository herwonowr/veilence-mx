import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type {
  Notification,
  NotificationChannel,
  NotificationRule,
} from "@/domains/notifications/types/notifications.types"

export const apiGetUnreadCount = async (): Promise<
  ApiResponse<{ count: number }>
> => fetchApi<{ count: number }>("/api/notifications/unread-count")

export const apiListNotifications = async (params?: {
  page?: number
  limit?: number
  unread?: boolean
}): Promise<ApiResponse<Notification[]>> => {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.set("page", String(params.page))
  if (params?.limit) searchParams.set("limit", String(params.limit))
  if (params?.unread) searchParams.set("unread", "true")
  const query = searchParams.toString()
  return fetchApi<Notification[]>(
    `/api/notifications${query ? `?${query}` : ""}`
  )
}

export const apiMarkNotificationRead = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/notifications/${id}/read`, { method: "PUT" })

export const apiMarkAllNotificationsRead = async (): Promise<
  ApiResponse<null>
> => fetchApi<null>("/api/notifications/read-all", { method: "PUT" })

export const apiListChannels = async (
  workspaceId: string
): Promise<ApiResponse<NotificationChannel[]>> =>
  fetchApi<NotificationChannel[]>(
    `/api/workspaces/${workspaceId}/notification-channels`
  )

export const apiCreateChannel = async (
  workspaceId: string,
  data: { name: string; type: string; config: string }
): Promise<ApiResponse<NotificationChannel>> =>
  fetchApi<NotificationChannel>(
    `/api/workspaces/${workspaceId}/notification-channels`,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  )

export const apiUpdateChannel = async (
  workspaceId: string,
  id: string,
  data: { name: string; config: string; isActive: boolean }
): Promise<ApiResponse<NotificationChannel>> =>
  fetchApi<NotificationChannel>(
    `/api/workspaces/${workspaceId}/notification-channels/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(data),
    }
  )

export const apiDeleteChannel = async (
  workspaceId: string,
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/notification-channels/${id}`, {
    method: "DELETE",
  })

export const testNotificationChannel = async (
  workspaceId: string,
  channelId: string
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>(
    `/api/workspaces/${workspaceId}/notification-channels/${channelId}/test`,
    { method: "POST" }
  )

export const apiListRules = async (
  workspaceId: string
): Promise<ApiResponse<NotificationRule[]>> =>
  fetchApi<NotificationRule[]>(
    `/api/workspaces/${workspaceId}/notification-rules`
  )

export const apiCreateRule = async (
  workspaceId: string,
  data: { channelId: string; severity: string }
): Promise<ApiResponse<NotificationRule>> =>
  fetchApi<NotificationRule>(
    `/api/workspaces/${workspaceId}/notification-rules`,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  )

export const apiDeleteNotification = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/notifications/${id}`, { method: "DELETE" })

export const apiDeleteAllNotifications = async (): Promise<
  ApiResponse<null>
> => fetchApi<null>("/api/notifications/all", { method: "DELETE" })

export const apiDeleteBatchNotifications = async (
  ids: string[]
): Promise<ApiResponse<null>> =>
  fetchApi<null>("/api/notifications", {
    method: "DELETE",
    body: JSON.stringify({ ids }),
  })

export const apiDeleteRule = async (
  workspaceId: string,
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/workspaces/${workspaceId}/notification-rules/${id}`, {
    method: "DELETE",
  })
