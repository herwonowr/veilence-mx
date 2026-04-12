"use client"

export {
  useUnreadCount,
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  notificationKeys,
} from "@/features/notifications/hooks/use-notifications"

export {
  useChannels,
  useCreateChannel,
  useUpdateChannel,
  useDeleteChannel,
  useTestChannel,
  useRules,
  useCreateRule,
  useDeleteRule,
  channelKeys,
  ruleKeys,
} from "@/features/notifications/hooks/use-channels"

export { NotificationBell } from "@/features/notifications/ui/notification-bell"
export { ChannelsView } from "@/features/notifications/ui/channels-view"
