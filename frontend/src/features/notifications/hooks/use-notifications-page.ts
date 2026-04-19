"use client"

import { useMemo, useState } from "react"
import { useSearchParams } from "next/navigation"
import { useFilterParams } from "@/core"
import {
  useNotifications,
  useUnreadCount,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useDeleteNotification,
  useDeleteAllNotifications,
  useDeleteBatchNotifications,
} from "@/features/notifications/hooks/use-notifications"
import {
  classifySeverity,
  EVENT_TYPE_LABELS,
} from "@/features/notifications/ui/notification-helpers"
import type { SeverityLevel } from "@/features/notifications/ui/notification-helpers"
import type { ActiveFilter } from "@/ui"

type PaginationState = {
  pageIndex: number
  pageSize: number
}

const VALID_READ_FILTERS = ["unread", "read", "all"] as const
const VALID_SEVERITY_FILTERS: ReadonlyArray<SeverityLevel> = ["critical", "high", "medium", "info"]
const VALID_EVENT_TYPES = Object.keys(EVENT_TYPE_LABELS)

export const useNotificationsPage = () => {
  const searchParams = useSearchParams()

  const initialRead = searchParams.get("read") ?? ""
  const initialSeverity = searchParams.get("severity") ?? ""
  const initialEventType = searchParams.get("eventType") ?? ""

  const [readFilter, setReadFilter] = useState<string>(
    (VALID_READ_FILTERS as readonly string[]).includes(initialRead) ? initialRead : "all",
  )
  const [severityFilter, setSeverityFilter] = useState<string>(
    (VALID_SEVERITY_FILTERS as readonly string[]).includes(initialSeverity) ? initialSeverity : "all",
  )
  const [eventTypeFilter, setEventTypeFilter] = useState<string>(
    VALID_EVENT_TYPES.includes(initialEventType) ? initialEventType : "all",
  )
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })

  // Wrap filter setters to reset pagination to first page on any filter change
  const setReadFilterAndResetPage = (value: string) => {
    setReadFilter(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }
  const setSeverityFilterAndResetPage = (value: string) => {
    setSeverityFilter(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }
  const setEventTypeFilterAndResetPage = (value: string) => {
    setEventTypeFilter(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  // Sync filter state → URL search params
  // "all" is the default for every filter - omit from URL when it matches
  const NOTIFICATIONS_FILTER_DEFAULTS = useMemo(() => ({
    read: "all",
    severity: "all",
    eventType: "all",
  }), [])
  useFilterParams(
    useMemo(() => ({
      read: readFilter,
      severity: severityFilter,
      eventType: eventTypeFilter,
    }), [readFilter, severityFilter, eventTypeFilter]),
    NOTIFICATIONS_FILTER_DEFAULTS,
  )

  const {
    data: notificationsRes,
    isLoading,
    isError,
    refetch,
  } = useNotifications({ limit: 500 })
  const allNotifications = useMemo(
    () => notificationsRes?.data ?? [],
    [notificationsRes?.data],
  )

  const { data: unreadRes } = useUnreadCount()
  const unreadCount = unreadRes?.data?.count ?? 0

  const markReadMutation = useMarkNotificationRead()
  const markAllReadMutation = useMarkAllNotificationsRead()
  const deleteMutation = useDeleteNotification()
  const deleteAllMutation = useDeleteAllNotifications()
  const deleteBatchMutation = useDeleteBatchNotifications()

  // Client-side filtering
  const filteredNotifications = useMemo(() => {
    let result = allNotifications

    if (readFilter === "unread") result = result.filter((n) => !n.isRead)
    if (readFilter === "read") result = result.filter((n) => n.isRead)

    if (severityFilter !== "all") {
      result = result.filter((n) => classifySeverity(n) === severityFilter)
    }

    if (eventTypeFilter !== "all") {
      result = result.filter((n) => n.eventType === eventTypeFilter)
    }

    return result
  }, [allNotifications, readFilter, severityFilter, eventTypeFilter])

  const total = filteredNotifications.length
  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  const paginatedNotifications = useMemo(() => {
    const start = pagination.pageIndex * pagination.pageSize
    return filteredNotifications.slice(start, start + pagination.pageSize)
  }, [filteredNotifications, pagination])

  const hasActiveFilters =
    readFilter !== "all" ||
    severityFilter !== "all" ||
    eventTypeFilter !== "all"

  const clearAllFilters = () => {
    setReadFilter("all")
    setSeverityFilter("all")
    setEventTypeFilter("all")
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  // Build active filters for FilterChips
  const activeFilters: ActiveFilter[] = [
    ...(readFilter !== "all"
      ? [
          {
            label: "Status",
            value: readFilter === "unread" ? "Unread" : "Read",
            onRemove: () => setReadFilterAndResetPage("all"),
          },
        ]
      : []),
    ...(severityFilter !== "all"
      ? [
          {
            label: "Severity",
            value:
              severityFilter.charAt(0).toUpperCase() +
              severityFilter.slice(1),
            onRemove: () => setSeverityFilterAndResetPage("all"),
          },
        ]
      : []),
    ...(eventTypeFilter !== "all"
      ? [
          {
            label: "Type",
            value: EVENT_TYPE_LABELS[eventTypeFilter] ?? eventTypeFilter,
            onRemove: () => setEventTypeFilterAndResetPage("all"),
          },
        ]
      : []),
  ]

  return {
    // Data
    notifications: paginatedNotifications,
    allNotificationsCount: allNotifications.length,
    total,
    unreadCount,
    isLoading,
    isError,

    // Filters
    readFilter,
    setReadFilter: setReadFilterAndResetPage,
    severityFilter,
    setSeverityFilter: setSeverityFilterAndResetPage,
    eventTypeFilter,
    setEventTypeFilter: setEventTypeFilterAndResetPage,
    hasActiveFilters,
    activeFilters,
    clearAllFilters,

    // Pagination
    pagination,
    setPagination,
    pageCount,

    // Actions
    markReadMutation,
    markAllReadMutation,
    deleteMutation,
    deleteAllMutation,
    deleteBatchMutation,
    refetch,
  }
}
