"use client"

import { useCallback, useMemo, useState } from "react"
import {
  Bell,
  CheckCheck,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
  RefreshCw,
  SearchX,
  Trash2,
  ListChecks,
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
import { Checkbox } from "@/ui/components/checkbox"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/components/alert-dialog"
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
    deleteMutation,
    deleteAllMutation,
    deleteBatchMutation,
    refetch,
  } = useNotificationsPage()

  // Multi-select state
  const [selectMode, setSelectMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [showDeleteAllDialog, setShowDeleteAllDialog] = useState(false)
  const [showDeleteSelectedDialog, setShowDeleteSelectedDialog] = useState(false)

  const rangeStart = total === 0 ? 0 : pagination.pageIndex * pagination.pageSize + 1
  const rangeEnd = Math.min(
    (pagination.pageIndex + 1) * pagination.pageSize,
    total,
  )

  const currentPageIds = useMemo(
    () => notifications.map((n) => n.id),
    [notifications],
  )

  const allPageSelected = useMemo(
    () => currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.has(id)),
    [currentPageIds, selectedIds],
  )

  const handleToggleSelect = useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const handleSelectAll = useCallback(() => {
    if (allPageSelected) {
      setSelectedIds((prev) => {
        const next = new Set(prev)
        for (const id of currentPageIds) {
          next.delete(id)
        }
        return next
      })
    } else {
      setSelectedIds((prev) => {
        const next = new Set(prev)
        for (const id of currentPageIds) {
          next.add(id)
        }
        return next
      })
    }
  }, [allPageSelected, currentPageIds])

  const handleExitSelectMode = useCallback(() => {
    setSelectMode(false)
    setSelectedIds(new Set())
  }, [])

  const handleMarkAllRead = useCallback(() => {
    markAllReadMutation.mutate()
  }, [markAllReadMutation])

  const handleDeleteAll = useCallback(() => {
    deleteAllMutation.mutate(undefined, {
      onSuccess: () => {
        setShowDeleteAllDialog(false)
        handleExitSelectMode()
      },
    })
  }, [deleteAllMutation, handleExitSelectMode])

  const handleDeleteSelected = useCallback(() => {
    const ids = Array.from(selectedIds)
    if (ids.length === 0) return
    deleteBatchMutation.mutate(ids, {
      onSuccess: () => {
        setShowDeleteSelectedDialog(false)
        setSelectedIds(new Set())
      },
    })
  }, [selectedIds, deleteBatchMutation])

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
          {selectMode ? (
            <>
              <span className="text-sm text-muted-foreground">
                {selectedIds.size} selected
              </span>
              <Button
                variant="destructive"
                size="sm"
                disabled={selectedIds.size === 0 || deleteBatchMutation.isPending}
                onClick={() => setShowDeleteSelectedDialog(true)}
              >
                <Trash2 className="mr-2 size-4" />
                Delete selected
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleExitSelectMode}
              >
                Cancel
              </Button>
            </>
          ) : (
            <>
              {allNotificationsCount > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setSelectMode(true)}
                >
                  <ListChecks className="mr-2 size-4" />
                  Select
                </Button>
              )}
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
              {allNotificationsCount > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowDeleteAllDialog(true)}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="mr-2 size-4" />
                  Delete all
                </Button>
              )}
            </>
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
              {/* Select all header row */}
              {selectMode && notifications.length > 0 && (
                <div className="flex items-center gap-3 border-b border-border px-4 py-2">
                  <div className="flex w-5 shrink-0 items-center justify-center">
                    <Checkbox
                      checked={allPageSelected}
                      onCheckedChange={handleSelectAll}
                      aria-label="Select all"
                    />
                  </div>
                </div>
              )}

              <ul className="divide-y divide-border">
                {notifications.map((notification) => (
                  <NotificationPageItem
                    key={notification.id}
                    notification={notification}
                    onMarkRead={(id) => markReadMutation.mutate(id)}
                    onDelete={(id) => deleteMutation.mutate(id)}
                    isMarkingRead={markReadMutation.isPending}
                    isDeleting={deleteMutation.isPending}
                    selectMode={selectMode}
                    isSelected={selectedIds.has(notification.id)}
                    onToggleSelect={handleToggleSelect}
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

      {/* Delete All Confirmation Dialog */}
      <AlertDialog open={showDeleteAllDialog} onOpenChange={setShowDeleteAllDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete all notifications?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete all {allNotificationsCount} notification
              {allNotificationsCount !== 1 ? "s" : ""}. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteAll}
              disabled={deleteAllMutation.isPending}
            >
              {deleteAllMutation.isPending ? "Deleting..." : "Delete all"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Selected Confirmation Dialog */}
      <AlertDialog open={showDeleteSelectedDialog} onOpenChange={setShowDeleteSelectedDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete selected notifications?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete {selectedIds.size} notification
              {selectedIds.size !== 1 ? "s" : ""}. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteSelected}
              disabled={deleteBatchMutation.isPending}
            >
              {deleteBatchMutation.isPending ? "Deleting..." : "Delete selected"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
