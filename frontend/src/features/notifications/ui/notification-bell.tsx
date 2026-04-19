"use client"

import { useCallback, useMemo, useState } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Bell, CheckCheck } from "lucide-react"
import { Button, buttonVariants, Skeleton, Badge, ScrollArea } from "@/ui"
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/ui"
import { cn } from "@/core"
import { useAuth } from "@/core"
import type { Notification } from "@/domains/notifications"
import {
  useUnreadCount,
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useDeleteNotification,
} from "@/features/notifications/hooks/use-notifications"
import { getNotificationLink } from "@/features/notifications/ui/notification-helpers"
import { NotificationItem } from "@/features/notifications/ui/notification-item"
import { NotificationEmptyState } from "@/features/notifications/ui/notification-empty"

// ---------------------------------------------------------------------------
// Loading skeletons
// ---------------------------------------------------------------------------

const LoadingSkeletons = () => (
  <div className="divide-y divide-border">
    {Array.from({ length: 4 }).map((_, i) => (
      <div key={i} className="flex gap-3 px-4 py-3">
        {/* Icon skeleton */}
        <Skeleton className="size-8 shrink-0 rounded-lg" />
        {/* Content skeleton */}
        <div className="flex flex-1 flex-col gap-2">
          <div className="flex items-start justify-between gap-2">
            <Skeleton className="h-4 w-3/5" />
            <Skeleton className="h-5 w-14 shrink-0 rounded-full" />
          </div>
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-1/3" />
        </div>
      </div>
    ))}
  </div>
)

// ---------------------------------------------------------------------------
// NotificationBell
// ---------------------------------------------------------------------------

export const NotificationBell = () => {
  const { isAuthenticated } = useAuth()
  const router = useRouter()
  const [open, setOpen] = useState(false)

  const { data: unreadRes } = useUnreadCount(isAuthenticated)
  const unreadCount = unreadRes?.data?.count ?? 0

  const { data: notificationsRes, isLoading } = useNotifications(
    { limit: 20, unread: true },
    { enabled: isAuthenticated && open },
  )
  // Belt-and-suspenders: filter client-side in case optimistic updates
  // mark items as read before the refetch replaces the cached list.
  const notifications = useMemo(
    () => (notificationsRes?.data ?? []).filter((n) => !n.isRead),
    [notificationsRes?.data],
  )

  const markReadMutation = useMarkNotificationRead()
  const markAllReadMutation = useMarkAllNotificationsRead()
  const deleteMutation = useDeleteNotification()

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen)
  }, [])

  const handleNotificationClick = useCallback(
    (notification: Notification) => {
      if (!notification.isRead) {
        markReadMutation.mutate(notification.id)
      }
      setOpen(false)
      router.push(getNotificationLink(notification))
    },
    [router, markReadMutation],
  )

  const handleMarkRead = useCallback(
    (id: number) => {
      markReadMutation.mutate(id)
    },
    [markReadMutation],
  )

  const handleDelete = useCallback(
    (id: number) => {
      deleteMutation.mutate(id)
    },
    [deleteMutation],
  )

  const handleMarkAllRead = useCallback(() => {
    markAllReadMutation.mutate()
  }, [markAllReadMutation])

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      {/* ---- Trigger ---- */}
      <PopoverTrigger
        render={
          <button
            type="button"
            className={cn(
              buttonVariants({ variant: "ghost", size: "icon" }),
              "relative",
            )}
            aria-label={
              unreadCount > 0
                ? `Notifications (${unreadCount} unread)`
                : "Notifications"
            }
          />
        }
      >
        <Bell className="size-4" aria-hidden="true" />
        {unreadCount > 0 && (
          <Badge
            variant="destructive"
            className="absolute -top-1 -right-1 flex size-[18px] items-center justify-center rounded-full p-0 text-[10px] font-semibold tabular-nums"
          >
            {unreadCount > 99 ? "99+" : unreadCount}
          </Badge>
        )}
        <span className="sr-only">Notifications</span>
      </PopoverTrigger>

      {/* ---- Content ---- */}
      <PopoverContent
        align="end"
        className="w-96 p-0"
        sideOffset={8}
      >
        {/* Header */}
        <PopoverHeader className="flex flex-row items-center justify-between border-b px-4 py-3">
          <PopoverTitle className="text-sm font-semibold">
            Notifications
          </PopoverTitle>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              size="xs"
              className="text-xs text-muted-foreground hover:text-foreground"
              onClick={handleMarkAllRead}
            >
              <CheckCheck className="size-3.5" />
              Mark all read
            </Button>
          )}
        </PopoverHeader>

        {/* Scrollable list */}
        <ScrollArea className="max-h-[420px]">
          {isLoading && notifications.length === 0 ? (
            <LoadingSkeletons />
          ) : notifications.length === 0 ? (
            <NotificationEmptyState />
          ) : (
            <ul className="divide-y divide-border">
              {notifications.map((notification) => (
                <NotificationItem
                  key={notification.id}
                  notification={notification}
                  onClick={handleNotificationClick}
                  onMarkRead={handleMarkRead}
                  onDelete={handleDelete}
                  isMarkingRead={markReadMutation.isPending}
                  isDeleting={deleteMutation.isPending}
                />
              ))}
            </ul>
          )}
        </ScrollArea>

        {/* Footer */}
        <div className="border-t px-4 py-2.5">
          <Link
            href={unreadCount > 0 ? "/notifications?read=unread" : "/notifications"}
            className={cn(
              buttonVariants({ variant: "ghost", size: "sm" }),
              "w-full justify-center text-xs text-muted-foreground hover:text-foreground",
            )}
            onClick={() => setOpen(false)}
          >
            View all notifications
          </Link>
        </div>
      </PopoverContent>
    </Popover>
  )
}
