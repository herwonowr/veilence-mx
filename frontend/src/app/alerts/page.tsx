"use client"

import { useEffect, useState, useMemo } from "react"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
import { RotateCcw } from "lucide-react"
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
  }, [severityFilter, statusFilter, sorting])

  const handleStatusChange = (id: number, status: string) => {
    updateMutation.mutate({ id, status })
  }

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
          <span className="text-sm">
            {new Date(row.original.createdAt).toLocaleString()}
          </span>
        ),
      },
      {
        id: "actions",
        header: "Actions",
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex gap-1">
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
          </div>
        ),
      },
    ],
    []
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
          <div className="flex items-center gap-4">
            <Select
              value={severityFilter || "all"}
              onValueChange={(v) => setSeverityFilter(v === "all" ? "" : (v ?? ""))}
            >
              <SelectTrigger className="w-40">
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
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Statuses</SelectItem>
                <SelectItem value="new">New</SelectItem>
                <SelectItem value="acknowledged">Acknowledged</SelectItem>
                <SelectItem value="resolved">Resolved</SelectItem>
              </SelectContent>
            </Select>
            {(severityFilter || statusFilter) && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
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
                  <TableCell colSpan={columns.length} className="text-center text-muted-foreground py-8">
                    No alerts found.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>

          <DataTablePagination table={table} total={total} />
        </CardContent>
      </Card>
    </div>
  )
}
