"use client"

import { Bell } from "lucide-react"

export const NotificationEmptyState = () => (
  <div className="flex flex-col items-center justify-center gap-2 py-12 text-center">
    <div className="flex size-10 items-center justify-center rounded-full bg-muted">
      <Bell className="size-5 text-muted-foreground" />
    </div>
    <p className="text-sm font-medium text-muted-foreground">
      No notifications
    </p>
    <p className="text-xs text-muted-foreground/70">
      You&apos;re all caught up
    </p>
  </div>
)
