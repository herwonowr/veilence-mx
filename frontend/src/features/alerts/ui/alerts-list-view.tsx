"use client"

import { Suspense, useEffect, useState, useMemo, useCallback } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useDebouncedValue, useSortParams, useFilterParams, useResponsiveColumns, useCurrentWorkspaceRole, hasMinimumRole, ROUTES, type ColumnBreakpoints } from "@/core"
import { Card, CardContent, CardHeader, Badge, Button, SearchInput, Label, TableSkeleton, TableError, TableEmptyState, FilterChips, DataTablePagination, SortableHeader, type SkeletonColumn, type ActiveFilter } from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import type { Alert } from "@/domains/alerts"
import type { AlertSeverity, AlertStatus } from "@/domains/common"
import Link from "next/link"
import { Bell, ShieldCheck } from "lucide-react"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import { useAlerts, useUpdateAlert } from "@/features/alerts/hooks/use-alerts"

const severityVariant = (s: AlertSeverity) => {
  if (s === "critical") return "destructive" as const
  if (s === "high") return "destructive" as const
  if (s === "medium") return "default" as const
  return "secondary" as const
}

const VALID_SEVERITIES: AlertSeverity[] = ["low", "medium", "high", "critical"]
const VALID_STATUSES: AlertStatus[] = ["new", "acknowledged", "resolved"]

export const AlertsListView = () => (
  <Suspense>
    <AlertsContent />
  </Suspense>
)

const AlertsContent = () => {
  const router = useRouter()
  const searchParams = useSearchParams()

  const initialSeverity = searchParams.get("severity") ?? ""
  const initialStatus = searchParams.get("status") ?? ""

  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [severityFilter, setSeverityFilter] = useState(
    VALID_SEVERITIES.includes(initialSeverity as AlertSeverity) ? initialSeverity : ""
  )
  const [statusFilter, setStatusFilter] = useState(
    VALID_STATUSES.includes(initialStatus as AlertStatus) ? initialStatus : ""
  )
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

  // Sync filter state → URL search params
  useFilterParams(
    useMemo(() => ({
      severity: severityFilter,
      status: statusFilter,
    }), [severityFilter, statusFilter]),
  )

  const alertColumnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    createdAt: "desktop",
    message: "tablet",
  }), [])
  const columnVisibility = useResponsiveColumns(alertColumnBreakpoints)

  const sort = sorting[0]
  const { data: alertsRes, isLoading, isFetching, isError, refetch } = useAlerts({
    severity: severityFilter || undefined,
    status: statusFilter || undefined,
    search: debouncedSearch || undefined,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const alerts = alertsRes?.data ?? []
  const total = alertsRes?.meta?.total ?? 0

  const updateMutation = useUpdateAlert()

  const { role } = useCurrentWorkspaceRole()
  /** Viewers cannot acknowledge or resolve alerts */
  const canTriage = hasMinimumRole(role, "member")

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [severityFilter, statusFilter, debouncedSearch, sorting])

  const hasActiveFilters = !!(search || severityFilter || statusFilter)

  const clearAllFilters = () => {
    setSearch("")
    setSeverityFilter("")
    setStatusFilter("")
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(severityFilter
      ? [{ label: "Severity", value: severityFilter.charAt(0).toUpperCase() + severityFilter.slice(1), onRemove: () => setSeverityFilter("") }]
      : []),
    ...(statusFilter
      ? [{ label: "Status", value: statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1), onRemove: () => setStatusFilter("") }]
      : []),
    ...(search
      ? [{ label: "Search", value: search, onRemove: () => setSearch("") }]
      : []),
  ]

  const handleStatusChange = useCallback(
    (id: string, status: string) => {
      updateMutation.mutate({ id, status })
    },
    [updateMutation]
  )

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-16", header: "Severity" },
    { width: "w-24", header: "Package" },
    { width: "w-48", header: "Message" },
    { width: "w-16", header: "Status" },
    { width: "w-24", header: "Created" },
    { width: "w-28", header: "" },
  ]

  const columns = useMemo<ColumnDef<Alert>[]>(
    () => [
      {
        accessorKey: "severity",
        header: ({ column }) => <SortableHeader column={column} title="Severity" />,
        cell: ({ row }) => (
          <Badge variant={severityVariant(row.original.severity)}>
            {row.original.severity}
          </Badge>
        ),
      },
      {
        accessorKey: "packageName",
        header: ({ column }) => <SortableHeader column={column} title="Package" />,
        cell: ({ row }) => (
          <Link
            href={ROUTES.PACKAGE_DETAIL(row.original.packageId)}
            className="font-medium hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            {row.original.packageName}
          </Link>
        ),
      },
      {
        accessorKey: "message",
        header: ({ column }) => <SortableHeader column={column} title="Message" />,
        cell: ({ row }) => (
          <span className="max-w-50 truncate block">{row.original.message}</span>
        ),
      },
      {
        accessorKey: "status",
        header: ({ column }) => <SortableHeader column={column} title="Status" />,
        cell: ({ row }) => (
          <Badge variant="outline">{row.original.status}</Badge>
        ),
      },
      {
        accessorKey: "createdAt",
        header: ({ column }) => <SortableHeader column={column} title="Created" />,
        cell: ({ row }) => (
          <span className="text-sm whitespace-nowrap">
            {new Date(row.original.createdAt).toLocaleString()}
          </span>
        ),
      },
      {
        id: "actions",
        header: () => <span className="sr-only">Actions</span>,
        enableSorting: false,
        meta: { headerClassName: "w-[1%] whitespace-nowrap text-right", cellClassName: "text-right" },
        cell: ({ row }) => (
          <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
            {canTriage && row.original.status === "new" && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleStatusChange(row.original.id, "acknowledged")}
              >
                <Bell className="size-3.5 text-emerald-600 dark:text-emerald-500 mr-1" />
                Acknowledge
              </Button>
            )}
            {canTriage && row.original.status !== "resolved" && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleStatusChange(row.original.id, "resolved")}
              >
                <ShieldCheck className="size-3.5 text-emerald-600 dark:text-emerald-500 mr-1" />
                Resolve
              </Button>
            )}
            {(row.original.releaseId ?? row.original.analysisId) ? (
              <Link href={ROUTES.RELEASE_DETAIL(row.original.releaseId ?? row.original.analysisId ?? "")}>
                <Button variant="ghost" size="sm" aria-label={`View release for alert ${row.original.id}`}>
                  View Release
                </Button>
              </Link>
            ) : (
              <Button variant="ghost" size="sm" disabled aria-label="No release linked">
                View Release
              </Button>
            )}
          </div>
        ),
      },
    ],
    [handleStatusChange, canTriage]
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
  const table = useReactTable({
    data: alerts,
    columns,
    pageCount,
    state: { pagination, sorting, columnVisibility },
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <h1 className="text-3xl font-bold">Alerts</h1>
        {!canTriage && role === "viewer" && (
          <Badge variant="outline" className="text-muted-foreground">
            Read-only
          </Badge>
        )}
      </div>

      <Card>
        <CardHeader>
          <div className="space-y-3">
            <div className="flex flex-wrap items-end gap-4">
              <SearchInput
                value={search}
                onChange={setSearch}
                onClear={() => setSearch("")}
                isLoading={isFetching && !!debouncedSearch}
                placeholder="Search by package name or message..."
                aria-label="Search alerts"
              />
              <div className="space-y-1">
                <Label htmlFor="alerts-severity-filter" className="text-xs text-muted-foreground">
                  Severity
                </Label>
                <Select
                  value={severityFilter || "all"}
                  onValueChange={(v) => setSeverityFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="alerts-severity-filter" className="w-40">
                    <SelectValue>{severityFilter ? severityFilter.charAt(0).toUpperCase() + severityFilter.slice(1) : "All Severities"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Severities</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                    <SelectItem value="high">High</SelectItem>
                    <SelectItem value="medium">Medium</SelectItem>
                    <SelectItem value="low">Low</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label htmlFor="alerts-status-filter" className="text-xs text-muted-foreground">
                  Status
                </Label>
                <Select
                  value={statusFilter || "all"}
                  onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="alerts-status-filter" className="w-40">
                    <SelectValue>{statusFilter ? statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1) : "All Statuses"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Statuses</SelectItem>
                    <SelectItem value="new">New</SelectItem>
                    <SelectItem value="acknowledged">Acknowledged</SelectItem>
                    <SelectItem value="resolved">Resolved</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            {hasActiveFilters && (
              <FilterChips filters={activeFilters} onClearAll={clearAllFilters} />
            )}
          </div>
        </CardHeader>
        <CardContent>
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
                    clickable
                    onClick={() => router.push(ROUTES.ALERT_DETAIL(row.original.id))}
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
                    icon={<Bell className="h-8 w-8" />}
                    title="No alerts match your filters."
                  >
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={clearAllFilters}
                    >
                      Clear Filters
                    </Button>
                  </TableEmptyState>
                ) : (
                  <TableEmptyState
                    colSpan={columns.length}
                    icon={<ShieldCheck className="h-8 w-8 text-green-600" />}
                    title="All clear!"
                    description="No alerts found. Your packages are looking safe."
                  >
                    <Link href={ROUTES.PACKAGES}>
                      <Button variant="outline" size="sm">
                        View Packages
                      </Button>
                    </Link>
                  </TableEmptyState>
                )
              )}
            </TableBody>
          </Table>
          )}
          </div>

          <DataTablePagination table={table} total={total} />
        </CardContent>
      </Card>
    </div>
  )
}
