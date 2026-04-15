"use client"

import { useCallback, useMemo, useState } from "react"
import { useRouter } from "next/navigation"
import { Bell, CheckCheck } from "lucide-react"
import { Button, buttonVariants } from "@/ui/components/button"
import { Skeleton } from "@/ui/components/skeleton"
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/ui/components/popover"
import { Badge } from "@/ui/components/badge"
import { cn } from "@/core/utils"
import { ScrollArea } from "@/ui/components/scroll-area"
import { useAuth } from "@/core/providers/auth-provider"
import type { Notification } from "@/domains/notifications"
import {
  useUnreadCount,
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
} from "@/features/notifications/hooks/use-notifications"

type SeverityLevel = "critical" | "high" | "medium" | "info"

const VALID_SEVERITIES: ReadonlyArray<SeverityLevel> = ["critical", "high", "medium", "info"]

const classifySeverity = (notification: Notification): SeverityLevel => {
  const mapped = notification.severity === "low" ? "info" : notification.severity
  return VALID_SEVERITIES.includes(mapped as SeverityLevel)
    ? (mapped as SeverityLevel)
    : "info"
}

const getNotificationLink = (notification: Notification): string => {
  switch (notification.referenceType) {
    case "alert":
      return notification.referenceId > 0
        ? `/alerts/${notification.referenceId}`
        : "/alerts"
    case "release":
      return notification.referenceId > 0
        ? `/releases/${notification.referenceId}`
        : "/releases"
    case "package":
      return "/packages"
    default:
      return "/dashboard"
  }
}

const severityVariant = (severity: SeverityLevel): "destructive" | "default" | "secondary" | "outline" => {
  if (severity === "critical") return "destructive"
  if (severity === "high") return "default"
  if (severity === "medium") return "secondary"
  return "outline"
}

const formatTimeAgo = (dateStr: string): string => {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const diffSec = Math.floor(diffMs / 1000)

  if (diffSec < 60) return "just now"
  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHr = Math.floor(diffMin / 60)
  if (diffHr < 24) return `${diffHr}h ago`
  const diffDay = Math.floor(diffHr / 24)
  if (diffDay < 30) return `${diffDay}d ago`
  return new Date(dateStr).toLocaleDateString()
}

export const NotificationBell = () => {
  const { isAuthenticated } = useAuth()
  const router = useRouter()
  const [open, setOpen] = useState(false)

  const { data: unreadRes } = useUnreadCount(isAuthenticated)
  const unreadCount = unreadRes?.data?.count ?? 0

  const { data: notificationsRes, isLoading } = useNotifications(
    { limit: 20 },
    { enabled: isAuthenticated && open }
  )
  const notifications = useMemo(
    () => notificationsRes?.data ?? [],
    [notificationsRes?.data]
  )

  const markReadMutation = useMarkNotificationRead()
  const markAllReadMutation = useMarkAllNotificationsRead()

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      setOpen(nextOpen)
    },
    []
  )

  const handleNotificationClick = useCallback(
    (notification: Notification) => {
      if (!notification.isRead) {
        markReadMutation.mutate(notification.id)
      }
      setOpen(false)
      router.push(getNotificationLink(notification))
    },
    [router, markReadMutation]
  )

  const handleMarkAllRead = useCallback(() => {
    markAllReadMutation.mutate()
  }, [markAllReadMutation])

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <button
            type="button"
            className={cn(
              buttonVariants({ variant: "ghost", size: "icon" }),
              "relative"
            )}
            aria-label={unreadCount > 0 ? `Notifications (${unreadCount} unread)` : "Notifications"}
          />
        }
      >
        <Bell className="size-4" aria-hidden="true" />
        {unreadCount > 0 && (
          <Badge
            variant="destructive"
            className="absolute -top-1 -right-1 size-5 items-center justify-center rounded-full p-0 text-[10px]"
          >
            {unreadCount > 99 ? "99+" : unreadCount}
          </Badge>
        )}
        <span className="sr-only">Notifications</span>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        className="w-80 p-0"
        sideOffset={8}
      >
        <PopoverHeader className="flex flex-row items-center justify-between border-b px-3 py-2">
          <PopoverTitle>Notifications</PopoverTitle>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="h-auto px-2 py-1 text-xs text-muted-foreground"
              onClick={handleMarkAllRead}
            >
              <CheckCheck className="mr-1 size-3" />
              Mark all read
            </Button>
          )}
        </PopoverHeader>

        <ScrollArea className="max-h-80">
          {isLoading && notifications.length === 0 ? (
            <div className="divide-y">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex flex-col gap-2 px-3 py-2.5">
                  <div className="flex items-start justify-between gap-2">
                    <Skeleton className="h-4 w-3/4" />
                    <Skeleton className="h-4 w-12 shrink-0 rounded-full" />
                  </div>
                  <Skeleton className="h-3 w-full" />
                  <Skeleton className="h-3 w-1/2" />
                </div>
              ))}
            </div>
          ) : notifications.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              No new notifications
            </div>
          ) : (
            <ul className="divide-y">
              {notifications.map((notification) => {
                const severity = classifySeverity(notification)
                return (
                <li key={notification.id}>
                  <Button
                    variant="ghost"
                    onClick={() => handleNotificationClick(notification)}
                    className={`flex w-full flex-col gap-1 px-3 py-2.5 text-left transition-colors hover:bg-muted/50 ${
                      notification.isRead ? "opacity-60" : ""
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <span className="text-sm font-medium leading-tight">
                        {notification.title}
                      </span>
                      <Badge
                        variant={severityVariant(severity)}
                        className="shrink-0 text-[10px]"
                      >
                        {severity}
                      </Badge>
                    </div>
                    <p className="line-clamp-2 text-xs text-muted-foreground">
                      {notification.message}
                    </p>
                    <div className="flex items-center gap-2">
                      <time className="text-[11px] text-muted-foreground/70">
                        {formatTimeAgo(notification.sentAt)}
                      </time>
                      {!notification.isRead && (
                        <span className="size-1.5 rounded-full bg-blue-500" />
                      )}
                    </div>
                  </Button>
                </li>
                )
              })}
            </ul>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  )
}
