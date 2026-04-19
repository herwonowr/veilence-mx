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
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/notifications/${id}/read`, { method: "PUT" })

export const apiMarkAllNotificationsRead = async (): Promise<
  ApiResponse<null>
> => fetchApi<null>("/api/notifications/read-all", { method: "PUT" })

export const apiListChannels = async (
  orgId: number
): Promise<ApiResponse<NotificationChannel[]>> =>
  fetchApi<NotificationChannel[]>(
    `/api/orgs/${orgId}/notification-channels`
  )

export const apiCreateChannel = async (
  orgId: number,
  data: { name: string; type: string; config: string }
): Promise<ApiResponse<NotificationChannel>> =>
  fetchApi<NotificationChannel>(
    `/api/orgs/${orgId}/notification-channels`,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  )

export const apiUpdateChannel = async (
  orgId: number,
  id: number,
  data: { name: string; config: string; isActive: boolean }
): Promise<ApiResponse<NotificationChannel>> =>
  fetchApi<NotificationChannel>(
    `/api/orgs/${orgId}/notification-channels/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(data),
    }
  )

export const apiDeleteChannel = async (
  orgId: number,
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/orgs/${orgId}/notification-channels/${id}`, {
    method: "DELETE",
  })

export const testNotificationChannel = async (
  orgId: number,
  channelId: number
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>(
    `/api/orgs/${orgId}/notification-channels/${channelId}/test`,
    { method: "POST" }
  )

export const apiListRules = async (
  orgId: number
): Promise<ApiResponse<NotificationRule[]>> =>
  fetchApi<NotificationRule[]>(
    `/api/orgs/${orgId}/notification-rules`
  )

export const apiCreateRule = async (
  orgId: number,
  data: { channelId: number; severity: string }
): Promise<ApiResponse<NotificationRule>> =>
  fetchApi<NotificationRule>(
    `/api/orgs/${orgId}/notification-rules`,
    {
      method: "POST",
      body: JSON.stringify(data),
    }
  )

export const apiDeleteNotification = async (
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/notifications/${id}`, { method: "DELETE" })

export const apiDeleteAllNotifications = async (): Promise<
  ApiResponse<null>
> => fetchApi<null>("/api/notifications/all", { method: "DELETE" })

export const apiDeleteBatchNotifications = async (
  ids: number[]
): Promise<ApiResponse<null>> =>
  fetchApi<null>("/api/notifications", {
    method: "DELETE",
    body: JSON.stringify({ ids }),
  })

export const apiDeleteRule = async (
  orgId: number,
  id: number
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/orgs/${orgId}/notification-rules/${id}`, {
    method: "DELETE",
  })
