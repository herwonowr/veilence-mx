export type NotificationChannelType = "email" | "slack" | "webhook"

export interface Notification {
  id: number
  orgId: number
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
  orgId: number
  name: string
  type: NotificationChannelType
  config: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface NotificationRule {
  id: number
  orgId: number
  channelId: number
  severity: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}
