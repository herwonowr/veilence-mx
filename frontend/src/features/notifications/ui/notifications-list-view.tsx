"use client"

import {
  Bell,
  CheckCheck,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
  RefreshCw,
  SearchX,
} from "lucide-react"
import { Button } from "@/ui/components/button"
import { Card, CardContent, CardHeader } from "@/ui/components/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import { Label } from "@/ui/components/label"
import { Skeleton } from "@/ui/components/skeleton"
import { EmptyState } from "@/ui/feedback/empty-state"
import { FilterChips } from "@/ui/data/filter-chips"
import { NotificationPageItem } from "@/features/notifications/ui/notification-page-item"
import { useNotificationsPage } from "@/features/notifications/hooks/use-notifications-page"

// ---------------------------------------------------------------------------
// Loading skeleton
// ---------------------------------------------------------------------------

const NotificationListSkeleton = () => (
  <div className="divide-y divide-border">
    {Array.from({ length: 8 }).map((_, i) => (
      <div key={i} className="flex gap-3 px-4 py-4">
        {/* Icon skeleton */}
        <Skeleton className="size-9 shrink-0 rounded-lg" />
        {/* Content skeleton */}
        <div className="flex flex-1 flex-col gap-2.5">
          <div className="flex items-start justify-between gap-2">
            <Skeleton className="h-4 w-2/5" />
            <div className="flex items-center gap-2">
              <Skeleton className="h-5 w-14 rounded-full" />
              <Skeleton className="h-3 w-12" />
            </div>
          </div>
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-4/5" />
          <Skeleton className="h-3 w-1/4" />
        </div>
      </div>
    ))}
  </div>
)

// ---------------------------------------------------------------------------
// NotificationsListView — full page component
// ---------------------------------------------------------------------------

export const NotificationsListView = () => {
  const {
    notifications,
    allNotificationsCount,
    total,
    unreadCount,
    isLoading,
    isError,
    readFilter,
    setReadFilter,
    severityFilter,
    setSeverityFilter,
    eventTypeFilter,
    setEventTypeFilter,
    hasActiveFilters,
    activeFilters,
    clearAllFilters,
    pagination,
    setPagination,
    pageCount,
    markReadMutation,
    markAllReadMutation,
    refetch,
  } = useNotificationsPage()

  const rangeStart = total === 0 ? 0 : pagination.pageIndex * pagination.pageSize + 1
  const rangeEnd = Math.min(
    (pagination.pageIndex + 1) * pagination.pageSize,
    total,
  )

  const handleMarkAllRead = () => {
    markAllReadMutation.mutate()
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-3xl font-bold">Notifications</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {allNotificationsCount} notification
            {allNotificationsCount !== 1 ? "s" : ""}
            {unreadCount > 0 && ` (${unreadCount} unread)`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {unreadCount > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleMarkAllRead}
              disabled={markAllReadMutation.isPending}
            >
              <CheckCheck className="mr-2 size-4" />
              Mark all read
            </Button>
          )}
        </div>
      </div>

      {/* Filter Bar + Notification List */}
      <Card>
        <CardHeader>
          <div className="space-y-3">
            <div className="flex flex-wrap items-end gap-4">
              {/* Read state filter */}
              <div className="space-y-1">
                <Label
                  htmlFor="notif-status-filter"
                  className="text-xs text-muted-foreground"
                >
                  Status
                </Label>
                <Select
                  value={readFilter}
                  onValueChange={(v) => {
                    if (v) setReadFilter(v)
                  }}
                >
                  <SelectTrigger id="notif-status-filter" className="w-28">
                    <SelectValue>
                      {readFilter === "unread"
                        ? "Unread"
                        : readFilter === "read"
                          ? "Read"
                          : "All"}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="unread">Unread</SelectItem>
                    <SelectItem value="read">Read</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Severity filter */}
              <div className="space-y-1">
                <Label
                  htmlFor="notif-severity-filter"
                  className="text-xs text-muted-foreground"
                >
                  Severity
                </Label>
                <Select
                  value={severityFilter}
                  onValueChange={(v) => {
                    if (v) setSeverityFilter(v)
                  }}
                >
                  <SelectTrigger id="notif-severity-filter" className="w-36">
                    <SelectValue>
                      {severityFilter === "all"
                        ? "All Severities"
                        : severityFilter.charAt(0).toUpperCase() +
                          severityFilter.slice(1)}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Severities</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                    <SelectItem value="high">High</SelectItem>
                    <SelectItem value="medium">Medium</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Event type filter */}
              <div className="space-y-1">
                <Label
                  htmlFor="notif-type-filter"
                  className="text-xs text-muted-foreground"
                >
                  Event Type
                </Label>
                <Select
                  value={eventTypeFilter}
                  onValueChange={(v) => {
                    if (v) setEventTypeFilter(v)
                  }}
                >
                  <SelectTrigger id="notif-type-filter" className="w-44">
                    <SelectValue>
                      {eventTypeFilter === "all"
                        ? "All Types"
                        : ({
                            "alert.created.malicious": "Malicious Alert",
                            "alert.created.suspicious": "Suspicious Alert",
                            "discovery.packages_added": "Package Discovery",
                            "analysis.error": "Analysis Error",
                            "diff.error": "Diff Error",
                            "packages.stale_removed": "Package Removed",
                          } as Record<string, string>)[eventTypeFilter] ??
                          eventTypeFilter}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Types</SelectItem>
                    <SelectItem value="alert.created.malicious">
                      Malicious Alert
                    </SelectItem>
                    <SelectItem value="alert.created.suspicious">
                      Suspicious Alert
                    </SelectItem>
                    <SelectItem value="discovery.packages_added">
                      Package Discovery
                    </SelectItem>
                    <SelectItem value="analysis.error">
                      Analysis Error
                    </SelectItem>
                    <SelectItem value="diff.error">Diff Error</SelectItem>
                    <SelectItem value="packages.stale_removed">
                      Package Removed
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Active filter chips */}
            {hasActiveFilters && (
              <FilterChips
                filters={activeFilters}
                onClearAll={clearAllFilters}
              />
            )}
          </div>
        </CardHeader>

        <CardContent>
          {/* Loading state */}
          {isLoading ? (
            <NotificationListSkeleton />
          ) : isError ? (
            /* Error state */
            <EmptyState
              icon={<AlertTriangle className="size-8" />}
              title="Failed to load notifications"
              description="Something went wrong. Please try again."
            >
              <Button
                variant="outline"
                size="sm"
                onClick={() => refetch()}
              >
                <RefreshCw className="mr-2 size-4" />
                Retry
              </Button>
            </EmptyState>
          ) : allNotificationsCount === 0 ? (
            /* Zero state — no notifications at all */
            <EmptyState
              icon={<Bell className="size-8" />}
              title="No notifications yet"
              description="Notifications will appear here when packages are discovered, releases are analyzed, or alerts are triggered."
            />
          ) : total === 0 && hasActiveFilters ? (
            /* Filtered empty state */
            <EmptyState
              icon={<SearchX className="size-8" />}
              title="No matching notifications"
              description="Try adjusting your filters to find what you're looking for."
            >
              <Button
                variant="outline"
                size="sm"
                onClick={clearAllFilters}
              >
                Clear all filters
              </Button>
            </EmptyState>
          ) : (
            /* Notification list */
            <>
              <ul className="divide-y divide-border">
                {notifications.map((notification) => (
                  <NotificationPageItem
                    key={notification.id}
                    notification={notification}
                    onMarkRead={(id) => markReadMutation.mutate(id)}
                    isMarkingRead={markReadMutation.isPending}
                  />
                ))}
              </ul>

              {/* Pagination */}
              {pageCount > 1 && (
                <div className="mt-4 flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
                  <p className="text-sm text-muted-foreground">
                    Showing {rangeStart}-{rangeEnd} of {total}
                  </p>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={pagination.pageIndex === 0}
                      onClick={() =>
                        setPagination((p) => ({
                          ...p,
                          pageIndex: p.pageIndex - 1,
                        }))
                      }
                    >
                      <ChevronLeft className="mr-1 size-4" />
                      <span className="hidden sm:inline">Previous</span>
                    </Button>
                    <span className="text-sm tabular-nums text-muted-foreground">
                      Page {pagination.pageIndex + 1} of {pageCount}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={pagination.pageIndex >= pageCount - 1}
                      onClick={() =>
                        setPagination((p) => ({
                          ...p,
                          pageIndex: p.pageIndex + 1,
                        }))
                      }
                    >
                      <span className="hidden sm:inline">Next</span>
                      <ChevronRight className="ml-1 size-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
