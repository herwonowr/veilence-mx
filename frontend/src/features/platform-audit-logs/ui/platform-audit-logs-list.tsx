"use client"

import { Suspense, useEffect, useState, useMemo } from "react"
import {
  useDebouncedValue,
  useSortParams,
  useFilterParams,
  useResponsiveColumns,
  type ColumnBreakpoints,
} from "@/core"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Badge,
  Button,
  Input,
  Label,
  TableSkeleton,
  TableError,
  TableEmptyState,
  FilterChips,
  DataTablePagination,
  DataTableColumnToggle,
  SortableHeader,
  AuditLogDetailDialog,
  Calendar,
  Popover,
  PopoverContent,
  PopoverTrigger,
  buttonVariants,
  type SkeletonColumn,
  type ActiveFilter,
} from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import { ScrollText, CalendarIcon, Filter } from "lucide-react"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
  type VisibilityState,
} from "@tanstack/react-table"
import "@/ui/data/table.types"
import { usePlatformAuditLogs } from "@/features/platform-audit-logs"
import type { PlatformAuditLog } from "@/domains/platform-admin"
import { cn } from "@/core"

const formatStartOfDay = (d: Date): string => {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, "0")
  const dd = String(d.getDate()).padStart(2, "0")
  return `${yyyy}-${mm}-${dd}T00:00:00Z`
}

const formatEndOfDay = (d: Date): string => {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, "0")
  const dd = String(d.getDate()).padStart(2, "0")
  return `${yyyy}-${mm}-${dd}T23:59:59Z`
}

export const PlatformAuditLogsList = () => (
  <Suspense>
    <PlatformAuditLogsContent />
  </Suspense>
)

const PlatformAuditLogsContent = () => {
  const [selectedLog, setSelectedLog] = useState<PlatformAuditLog | null>(null)
  const [action, setAction] = useState("")
  const [resource, setResource] = useState("")
  const [userId, setUserId] = useState("")
  const [workspaceId, setWorkspaceId] = useState("")
  const debouncedAction = useDebouncedValue(action, 300)
  const debouncedResource = useDebouncedValue(resource, 300)
  const debouncedUserId = useDebouncedValue(userId, 300)
  const debouncedWorkspaceId = useDebouncedValue(workspaceId, 300)

  const [fromDate, setFromDate] = useState<Date | undefined>(undefined)
  const [toDate, setToDate] = useState<Date | undefined>(undefined)
  const [fromOpen, setFromOpen] = useState(false)
  const [toOpen, setToOpen] = useState(false)

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

  useFilterParams(
    useMemo(() => ({
      action: debouncedAction,
      resource: debouncedResource,
      user_email: debouncedUserId,
      workspace_name: debouncedWorkspaceId,
      from: fromDate ? formatStartOfDay(fromDate) : "",
      to: toDate ? formatEndOfDay(toDate) : "",
    }), [debouncedAction, debouncedResource, debouncedUserId, debouncedWorkspaceId, fromDate, toDate]),
  )

  const columnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    userAgent: "desktop",
    correlationId: "desktop",
    resourceId: "desktop",
  }), [])
  const responsiveVisibility = useResponsiveColumns(columnBreakpoints)
  const [userColumnVisibility, setUserColumnVisibility] = useState<VisibilityState>({
    details: false,
  })
  const columnVisibility = useMemo<VisibilityState>(
    () => ({ ...userColumnVisibility, ...responsiveVisibility }),
    [userColumnVisibility, responsiveVisibility],
  )

  const sort = sorting[0]
  const { data: logsRes, isLoading, isError, refetch } = usePlatformAuditLogs({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    action: debouncedAction || undefined,
    resource: debouncedResource || undefined,
    user_email: debouncedUserId || undefined,
    workspace_name: debouncedWorkspaceId || undefined,
    from_date: fromDate ? formatStartOfDay(fromDate) : undefined,
    to_date: toDate ? formatEndOfDay(toDate) : undefined,
    sort_by: sort?.id,
    sort_dir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const logs = logsRes?.data ?? []
  const total = logsRes?.meta?.total ?? 0

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [debouncedAction, debouncedResource, debouncedUserId, debouncedWorkspaceId, fromDate, toDate, sorting])

  const hasActiveFilters = !!(action || resource || userId || workspaceId || fromDate || toDate)

  const clearAllFilters = () => {
    setAction("")
    setResource("")
    setUserId("")
    setWorkspaceId("")
    setFromDate(undefined)
    setToDate(undefined)
    setSorting([])
  }

  const handleFromSelect = (date: Date | undefined) => {
    setFromDate(date)
    setFromOpen(false)
  }

  const handleToSelect = (date: Date | undefined) => {
    setToDate(date)
    setToOpen(false)
  }

  const activeFilters: ActiveFilter[] = [
    ...(action
      ? [{ label: "Action", value: action, onRemove: () => setAction("") }]
      : []),
    ...(resource
      ? [{ label: "Resource", value: resource, onRemove: () => setResource("") }]
      : []),
    ...(userId
      ? [{ label: "User", value: userId, onRemove: () => setUserId("") }]
      : []),
    ...(workspaceId
      ? [{ label: "Workspace", value: workspaceId, onRemove: () => setWorkspaceId("") }]
      : []),
    ...(fromDate
      ? [{ label: "From", value: fromDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }), onRemove: () => { setFromDate(undefined); setToDate(undefined) } }]
      : []),
    ...(toDate
      ? [{ label: "To", value: toDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }), onRemove: () => setToDate(undefined) }]
      : []),
  ]

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Timestamp" },
    { width: "w-24", header: "User" },
    { width: "w-24", header: "Workspace" },
    { width: "w-16", header: "Action" },
    { width: "w-20", header: "Resource" },
    { width: "w-40", header: "Details" },
    { width: "w-24", header: "IP Address" },
  ]

  const columns = useMemo<ColumnDef<PlatformAuditLog>[]>(
    () => [
      {
        accessorKey: "createdAt",
        header: ({ column }) => <SortableHeader column={column} title="Timestamp" />,
        enableHiding: false,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground whitespace-nowrap">
            {new Date(row.original.createdAt).toLocaleString()}
          </span>
        ),
      },
      {
        accessorKey: "userEmail",
        header: ({ column }) => <SortableHeader column={column} title="User" />,
        enableHiding: false,
        cell: ({ row }) => (
          <span className="text-sm">{row.original.userEmail || "-"}</span>
        ),
      },
      {
        accessorKey: "workspaceId",
        header: ({ column }) => <SortableHeader column={column} title="Workspace" />,
        cell: ({ row }) => (
          <span className="text-sm">
            {row.original.workspaceName || row.original.workspaceId || "-"}
          </span>
        ),
      },
      {
        accessorKey: "action",
        header: ({ column }) => <SortableHeader column={column} title="Action" />,
        enableHiding: false,
        cell: ({ row }) => (
          <Badge variant="outline">{row.original.action}</Badge>
        ),
      },
      {
        accessorKey: "resource",
        header: ({ column }) => <SortableHeader column={column} title="Resource" />,
        enableHiding: false,
        cell: ({ row }) => (
          <Badge variant="secondary" className="w-fit capitalize">
            {row.original.resource.replace(/_/g, " ")}
          </Badge>
        ),
      },
      {
        accessorKey: "resourceId",
        header: "Resource ID",
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.resourceId ? row.original.resourceId.slice(0, 8) : "-"}
          </span>
        ),
      },
      {
        accessorKey: "details",
        header: "Details",
        enableSorting: false,
        cell: ({ row }) => (
          <span className="max-w-sm text-sm truncate block">
            {row.original.details || "-"}
          </span>
        ),
      },
      {
        accessorKey: "ipAddress",
        header: "IP Address",
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.ipAddress || "-"}
          </span>
        ),
      },
    ],
    []
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
  const table = useReactTable({
    data: logs,
    columns,
    pageCount,
    state: { pagination, sorting, columnVisibility },
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    onColumnVisibilityChange: setUserColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
  })

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base flex items-center gap-2">
            <Filter className="size-4" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label htmlFor="filter-action" className="text-xs text-muted-foreground">
                  Action
                </Label>
                <Input
                  id="filter-action"
                  placeholder="e.g. create, update"
                  value={action}
                  onChange={(e) => setAction(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="filter-resource" className="text-xs text-muted-foreground">
                  Resource
                </Label>
                <Input
                  id="filter-resource"
                  placeholder="e.g. package, member"
                  value={resource}
                  onChange={(e) => setResource(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="filter-user-email" className="text-xs text-muted-foreground">
                  User Email
                </Label>
                <Input
                  id="filter-user-email"
                  placeholder="e.g. admin@example.com"
                  value={userId}
                  onChange={(e) => setUserId(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="filter-workspace-name" className="text-xs text-muted-foreground">
                  Workspace
                </Label>
                <Input
                  id="filter-workspace-name"
                  placeholder="e.g. Acme Corp"
                  value={workspaceId}
                  onChange={(e) => setWorkspaceId(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">From</Label>
                <Popover open={fromOpen} onOpenChange={setFromOpen}>
                  <PopoverTrigger
                    render={
                      <button
                        type="button"
                        className={cn(
                          buttonVariants({ variant: "outline" }),
                          "w-full justify-start text-left font-normal h-8"
                        )}
                        aria-label="Select start date"
                      />
                    }
                  >
                    <CalendarIcon className="mr-2 h-4 w-4 text-muted-foreground" />
                    {fromDate ? (
                      fromDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
                    ) : (
                      <span className="text-muted-foreground">Pick a date</span>
                    )}
                  </PopoverTrigger>
                  <PopoverContent align="start" className="w-auto">
                    <Calendar
                      mode="single"
                      selected={fromDate}
                      onSelect={handleFromSelect}
                      disabled={(date) => toDate ? date > toDate : date > new Date()}
                      defaultMonth={fromDate}
                    />
                  </PopoverContent>
                </Popover>
              </div>
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">To</Label>
                <Popover open={toOpen} onOpenChange={fromDate ? setToOpen : undefined}>
                  <PopoverTrigger
                    render={
                      <button
                        type="button"
                        disabled={!fromDate}
                        className={cn(
                          buttonVariants({ variant: "outline" }),
                          "w-full justify-start text-left font-normal h-8",
                          !fromDate && "opacity-50 cursor-not-allowed"
                        )}
                        aria-label="Select end date"
                      />
                    }
                  >
                    <CalendarIcon className="mr-2 h-4 w-4 text-muted-foreground" />
                    {toDate ? (
                      toDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
                    ) : (
                      <span className="text-muted-foreground">Pick a date</span>
                    )}
                  </PopoverTrigger>
                  <PopoverContent align="start" className="w-auto">
                    <Calendar
                      mode="single"
                      selected={toDate}
                      onSelect={handleToSelect}
                      disabled={(date) => fromDate ? date < fromDate || date > new Date() : date > new Date()}
                      defaultMonth={toDate ?? fromDate}
                    />
                  </PopoverContent>
                </Popover>
              </div>
            </div>
            {hasActiveFilters && (
              <FilterChips filters={activeFilters} onClearAll={clearAllFilters} />
            )}
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardContent>
          <div className="flex justify-end mb-3">
            <DataTableColumnToggle table={table} />
          </div>
          <div className="overflow-x-auto">
            {isLoading ? (
              <TableSkeleton columns={skeletonColumns} rows={5} />
            ) : isError ? (
              <TableError colSpan={columns.length} onRetry={() => refetch()} />
            ) : (
              <Table>
                <TableHeader>
                  {table.getHeaderGroups().map((headerGroup) => (
                    <TableRow key={headerGroup.id}>
                      {headerGroup.headers.map((header) => {
                        const sorted = header.column.getIsSorted()
                        return (
                          <TableHead
                            key={header.id}
                            className={header.column.columnDef.meta?.headerClassName}
                            aria-sort={sorted === "asc" ? "ascending" : sorted === "desc" ? "descending" : undefined}
                          >
                            {header.isPlaceholder
                              ? null
                              : flexRender(header.column.columnDef.header, header.getContext())}
                          </TableHead>
                        )
                      })}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows.length ? (
                    table.getRowModel().rows.map((row) => (
                      <TableRow
                        key={row.id}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => setSelectedLog(row.original)}
                      >
                        {row.getVisibleCells().map((cell) => (
                          <TableCell key={cell.id} className={cell.column.columnDef.meta?.cellClassName}>
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : (
                    hasActiveFilters ? (
                      <TableEmptyState
                        colSpan={columns.length}
                        icon={<ScrollText className="h-8 w-8" />}
                        title="No matching audit logs."
                        description="Try adjusting your filters."
                      >
                        <Button variant="outline" size="sm" onClick={clearAllFilters}>
                          Clear filters
                        </Button>
                      </TableEmptyState>
                    ) : (
                      <TableEmptyState
                        colSpan={columns.length}
                        icon={<ScrollText className="h-8 w-8" />}
                        title="No audit logs found."
                        description="Activity history across all workspaces will appear here."
                      />
                    )
                  )}
                </TableBody>
              </Table>
            )}
          </div>

          <DataTablePagination table={table} total={total} />
        </CardContent>
      </Card>

      <AuditLogDetailDialog
        log={selectedLog}
        onOpenChange={(open) => { if (!open) setSelectedLog(null) }}
      />
    </div>
  )
}
