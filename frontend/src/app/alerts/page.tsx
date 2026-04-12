"use client"

import { Suspense, useEffect, useState, useMemo, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import { useSortParams } from "@/hooks/use-sort-params"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { SearchInput } from "@/components/search-input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { Alert, AlertSeverity } from "@/types"
import Link from "next/link"
import { Bell, ShieldCheck } from "lucide-react"
import { TableSkeleton, type SkeletonColumn } from "@/components/table-skeleton"
import { TableError } from "@/components/table-error"
import { TableEmptyState } from "@/components/empty-state"
import { FilterChips, type ActiveFilter } from "@/components/filter-chips"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
  type SortingState,
} from "@tanstack/react-table"
import { DataTablePagination } from "@/components/data-table-pagination"
import { SortableHeader } from "@/components/sortable-header"
import { useResponsiveColumns, type ColumnBreakpoints } from "@/hooks/use-responsive-columns"
import { ProtectedRoute } from "@/components/protected-route"
import { useAlerts, useUpdateAlert } from "@/features/alerts"

function severityVariant(s: AlertSeverity) {
  if (s === "critical") return "destructive" as const
  if (s === "high") return "destructive" as const
  if (s === "medium") return "default" as const
  return "secondary" as const
}

export default function AlertsPage() {
  return (
    <ProtectedRoute>
      <Suspense>
        <AlertsContent />
      </Suspense>
    </ProtectedRoute>
  )
}

function AlertsContent() {
  const router = useRouter()
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [severityFilter, setSeverityFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

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
    (id: number, status: string) => {
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
    { width: "w-28", header: "Actions" },
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
            href={`/packages/${row.original.packageId}`}
            className="font-medium hover:underline"
          >
            {row.original.packageName}
          </Link>
        ),
      },
      {
        accessorKey: "message",
        header: ({ column }) => <SortableHeader column={column} title="Message" />,
        cell: ({ row }) => (
          <span className="max-w-[200px] truncate block">{row.original.message}</span>
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
        header: "Actions",
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex flex-wrap gap-1">
            {row.original.status === "new" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleStatusChange(row.original.id, "acknowledged")}
              >
                Acknowledge
              </Button>
            )}
            {row.original.status !== "resolved" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleStatusChange(row.original.id, "resolved")}
              >
                Resolve
              </Button>
            )}
            {/* Alert → Release deep link (prefer releaseId, fallback to analysisId) */}
            {(row.original.releaseId ?? row.original.analysisId) ? (
              <Link href={`/releases/${row.original.releaseId ?? row.original.analysisId}`}>
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
    [handleStatusChange]
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

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
      <h1 className="text-3xl font-bold">Alerts</h1>

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
                <label htmlFor="alerts-severity-filter" className="text-xs font-medium text-muted-foreground">
                  Severity
                </label>
                <Select
                  value={severityFilter || "all"}
                  onValueChange={(v) => setSeverityFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="alerts-severity-filter" className="w-40">
                    <SelectValue placeholder="All Severities" />
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
                <label htmlFor="alerts-status-filter" className="text-xs font-medium text-muted-foreground">
                  Status
                </label>
                <Select
                  value={statusFilter || "all"}
                  onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="alerts-status-filter" className="w-40">
                    <SelectValue placeholder="All Statuses" />
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
                    onClick={() => router.push(`/alerts/${row.original.id}`)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
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
                    <Link href="/packages">
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
