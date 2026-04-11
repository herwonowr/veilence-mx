"use client"

import { useEffect, useState, useMemo, useCallback } from "react"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
import { RotateCcw, Bell, ShieldCheck } from "lucide-react"
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
      <AlertsContent />
    </ProtectedRoute>
  )
}

function AlertsContent() {
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [severityFilter, setSeverityFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useState<SortingState>([])

  const sort = sorting[0]
  const { data: alertsRes } = useAlerts({
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

  const handleStatusChange = useCallback(
    (id: number, status: string) => {
      updateMutation.mutate({ id, status })
    },
    [updateMutation]
  )

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
          <span className="max-w-md truncate block">{row.original.message}</span>
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
    state: { pagination, sorting },
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
          <div className="flex flex-wrap items-center gap-4">
            <Input
              placeholder="Search by package name or message..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="max-w-xs"
              aria-label="Search alerts"
            />
            <Select
              value={severityFilter || "all"}
              onValueChange={(v) => setSeverityFilter(v === "all" ? "" : (v ?? ""))}
            >
              <SelectTrigger className="w-40" aria-label="Filter by severity">
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
            <Select
              value={statusFilter || "all"}
              onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
            >
              <SelectTrigger className="w-40" aria-label="Filter by status">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Statuses</SelectItem>
                <SelectItem value="new">New</SelectItem>
                <SelectItem value="acknowledged">Acknowledged</SelectItem>
                <SelectItem value="resolved">Resolved</SelectItem>
              </SelectContent>
            </Select>
            {(search || severityFilter || statusFilter) && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSearch("")
                  setSeverityFilter("")
                  setStatusFilter("")
                  setSorting([])
                }}
              >
                <RotateCcw className="mr-1 h-3 w-3" />
                Reset
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="text-center py-8">
                    <div className="flex flex-col items-center gap-3">
                      {search || severityFilter || statusFilter ? (
                        <>
                          <Bell className="h-8 w-8 text-muted-foreground" />
                          <p className="text-muted-foreground">No alerts match your filters.</p>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              setSearch("")
                              setSeverityFilter("")
                              setStatusFilter("")
                            }}
                          >
                            Clear Filters
                          </Button>
                        </>
                      ) : (
                        <>
                          <ShieldCheck className="h-8 w-8 text-green-600" />
                          <p className="text-muted-foreground font-medium">All clear!</p>
                          <p className="text-sm text-muted-foreground">No alerts found. Your packages are looking safe.</p>
                          <Link href="/packages">
                            <Button variant="outline" size="sm">
                              View Packages
                            </Button>
                          </Link>
                        </>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          </div>

          <DataTablePagination table={table} total={total} />
        </CardContent>
      </Card>
    </div>
  )
}
