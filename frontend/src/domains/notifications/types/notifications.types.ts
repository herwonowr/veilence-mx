export type NotificationChannelType = "email" | "slack" | "webhook"

export interface Notification {
  id: number
  workspaceId: number
  userId: number
  channelId: number
  severity: string
  eventType: string
  referenceId: number
  referenceType: string
  title: string
  message: string
  isRead: boolean
  sentAt: string
  createdAt: string
}

export interface NotificationChannel {
  id: number
  workspaceId: number
  name: string
  type: NotificationChannelType
  config: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface NotificationRule {
  id: number
  workspaceId: number
  channelId: number
  severity: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}
