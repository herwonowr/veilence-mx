"use client"

import Link from "next/link"
import { CheckCheck, Circle, Trash2 } from "lucide-react"
import { Badge } from "@/ui/components/badge"
import { Button } from "@/ui/components/button"
import { Checkbox } from "@/ui/components/checkbox"
import { cn } from "@/core/utils"
import type { Notification } from "@/domains/notifications"
import {
  classifySeverity,
  formatTimeAgo,
  getEventTypeConfig,
  getNotificationLink,
  SEVERITY_STYLES,
} from "@/features/notifications/ui/notification-helpers"

// ---------------------------------------------------------------------------
// Severity badge (shared visual atom)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// NotificationPageItem — full-width item for the /notifications page
// ---------------------------------------------------------------------------

type NotificationPageItemProps = {
  notification: Notification
  onMarkRead: (id: number) => void
  onDelete: (id: number) => void
  isMarkingRead: boolean
  isDeleting: boolean
  isSelected?: boolean
  onToggleSelect?: (id: number) => void
  selectMode?: boolean
}

export const NotificationPageItem = ({
  notification,
  onMarkRead,
  onDelete,
  isMarkingRead,
  isDeleting,
  isSelected = false,
  onToggleSelect,
  selectMode = false,
}: NotificationPageItemProps) => {
  const severity = classifySeverity(notification)
  const eventConfig = getEventTypeConfig(notification.eventType)
  const Icon = eventConfig.icon

  return (
    <li>
      <div
        className={cn(
          "group flex gap-3 px-4 py-4 transition-colors",
          "hover:bg-muted/50",
          !notification.isRead && "bg-muted/30",
          isSelected && "bg-primary/5",
        )}
      >
        {/* Checkbox for multi-select */}
        {selectMode && (
          <div className="flex shrink-0 items-center pt-1">
            <Checkbox
              checked={isSelected}
              onCheckedChange={() => onToggleSelect?.(notification.id)}
              aria-label={`Select notification: ${notification.title}`}
            />
          </div>
        )}
        {/* Left: Event type icon */}
        <div
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-lg",
            eventConfig.bgClass,
          )}
        >
          <Icon className={cn("size-4.5", eventConfig.textClass)} />
        </div>

        {/* Center: Content — expanded */}
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          {/* Row 1: Title + Severity badge + Timestamp */}
          <div className="flex items-start justify-between gap-2">
            <Link
              href={getNotificationLink(notification)}
              className={cn(
                "text-sm leading-tight hover:underline",
                notification.isRead ? "font-normal" : "font-medium",
              )}
            >
              {notification.title}
            </Link>
            <div className="flex shrink-0 items-center gap-2">
              <SeverityBadge severity={severity} />
              <time className="whitespace-nowrap text-xs text-muted-foreground/70">
                {formatTimeAgo(notification.sentAt)}
              </time>
            </div>
          </div>

          {/* Row 2: Full message — NO line-clamp */}
          <p className="text-sm text-muted-foreground">
            {notification.message}
          </p>

          {/* Row 3: Unread dot + Action buttons */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              {!notification.isRead && (
                <span
                  className="size-1.5 rounded-full bg-primary"
                  aria-label="Unread"
                />
              )}
            </div>
            <div
              className={cn(
                "flex items-center gap-1 transition-opacity",
                "opacity-0 group-hover:opacity-100 focus-within:opacity-100 max-sm:opacity-100",
              )}
            >
              {!notification.isRead && (
                <Button
                  variant="ghost"
                  size="xs"
                  onClick={() => onMarkRead(notification.id)}
                  disabled={isMarkingRead}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  <CheckCheck className="size-3.5" />
                  Mark read
                </Button>
              )}
              {notification.isRead && (
                <span className="flex items-center gap-1 px-2 text-xs text-muted-foreground/50">
                  <Circle className="size-3" />
                  Read
                </span>
              )}
              <Button
                variant="ghost"
                size="xs"
                onClick={() => onDelete(notification.id)}
                disabled={isDeleting}
                className="text-xs text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="size-3.5" />
                Delete
              </Button>
            </div>
          </div>
        </div>
      </div>
    </li>
  )
}
