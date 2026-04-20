export type NotificationChannelType = "email" | "slack" | "webhook"

export interface Notification {
  id: string
  workspaceId: string
  userId: string
  channelId: string
  severity: string
  eventType: string
  referenceId: string
  referenceType: string
  title: string
  message: string
  isRead: boolean
  sentAt: string
  createdAt: string
}

export interface NotificationChannel {
  id: string
  workspaceId: string
  name: string
  type: NotificationChannelType
  config: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface NotificationRule {
  id: string
  workspaceId: string
  channelId: string
  severity: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}
