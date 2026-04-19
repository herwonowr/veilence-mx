"use client"

import type { KeyboardEvent, MouseEvent } from "react"
import { Check, Trash2 } from "lucide-react"
import { Badge } from "@/ui/components/badge"
import { cn } from "@/core/utils"
import type { Notification } from "@/domains/notifications"
import {
  classifySeverity,
  formatTimeAgo,
  getEventTypeConfig,
  SEVERITY_STYLES,
} from "@/features/notifications/ui/notification-helpers"

type NotificationItemProps = {
  notification: Notification
  onClick: (notification: Notification) => void
  onMarkRead?: (id: number) => void
  onDelete?: (id: number) => void
  isMarkingRead?: boolean
  isDeleting?: boolean
}

const SeverityBadge = ({ severity }: { severity: string }) => {
  const style = SEVERITY_STYLES[severity as keyof typeof SEVERITY_STYLES]
  if (!style) return null

  return (
    <Badge
      className={cn(
        "shrink-0 border text-[10px] capitalize",
        style,
      )}
    >
      {severity}
    </Badge>
  )
}

export const NotificationItem = ({
  notification,
  onClick,
  onMarkRead,
  onDelete,
  isMarkingRead,
  isDeleting,
}: NotificationItemProps) => {
  const severity = classifySeverity(notification)
  const eventConfig = getEventTypeConfig(notification.eventType)
  const Icon = eventConfig.icon

  const handleMarkRead = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation()
    onMarkRead?.(notification.id)
  }

  const handleDelete = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation()
    onDelete?.(notification.id)
  }

  const handleOuterKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault()
      onClick(notification)
    }
  }

  return (
    <li>
      <div
        role="button"
        tabIndex={0}
        onClick={() => onClick(notification)}
        onKeyDown={handleOuterKeyDown}
        className={cn(
          "group flex w-full gap-3 px-4 py-3 text-left transition-colors",
          "hover:bg-muted/50 focus-visible:bg-muted/50 focus-visible:outline-none",
          !notification.isRead && "bg-muted/30",
        )}
      >
        {/* Left: Event type icon */}
        <div
          className={cn(
            "flex size-8 shrink-0 items-center justify-center rounded-lg",
            eventConfig.bgClass,
          )}
        >
          <Icon className={cn("size-4", eventConfig.textClass)} />
        </div>

        {/* Center: Content */}
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          {/* Row 1: Title + Severity badge */}
          <div className="flex items-start justify-between gap-2">
            <span
              className={cn(
                "text-sm leading-tight",
                notification.isRead ? "font-normal" : "font-medium",
              )}
            >
              {notification.title}
            </span>
            <div className="flex shrink-0 items-center gap-1.5">
              <SeverityBadge severity={severity} />
              {/* Mark as read button - visible on hover for unread items */}
              {!notification.isRead && onMarkRead && (
                <button
                  type="button"
                  onClick={handleMarkRead}
                  disabled={isMarkingRead}
                  className={cn(
                    "flex size-5 items-center justify-center rounded-full transition-all",
                    "text-muted-foreground hover:bg-primary/10 hover:text-primary",
                    "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
                    "disabled:pointer-events-none disabled:opacity-50",
                  )}
                  aria-label="Mark as read"
                >
                  <Check className="size-3" />
                </button>
              )}
              {onDelete && (
                <button
                  type="button"
                  onClick={handleDelete}
                  disabled={isDeleting}
                  className={cn(
                    "flex size-5 items-center justify-center rounded-full transition-all",
                    "text-muted-foreground hover:bg-destructive/10 hover:text-destructive",
                    "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
                    "disabled:pointer-events-none disabled:opacity-50",
                  )}
                  aria-label="Delete notification"
                >
                  <Trash2 className="size-3" />
                </button>
              )}
            </div>
          </div>

          {/* Row 2: Message preview */}
          <p className="line-clamp-2 text-xs text-muted-foreground">
            {notification.message}
          </p>

          {/* Row 3: Timestamp + unread dot */}
          <div className="flex items-center gap-1.5">
            <time className="text-[11px] text-muted-foreground/70">
              {formatTimeAgo(notification.sentAt)}
            </time>
            {!notification.isRead && (
              <span
                className="size-1.5 rounded-full bg-primary"
                aria-label="Unread"
              />
            )}
          </div>
        </div>
      </div>
    </li>
  )
}
