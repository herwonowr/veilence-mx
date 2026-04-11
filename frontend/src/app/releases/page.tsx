"use client"

import { useEffect, useState, useMemo } from "react"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import Link from "next/link"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { RecentRelease, Classification } from "@/types"
import { Skeleton } from "@/components/ui/skeleton"
import { CheckCircle, RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
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
import { useRecentReleases } from "@/features/dashboard"

function classificationVariant(c: Classification) {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  if (c === "baseline") return "outline" as const
  return "secondary" as const
}

export default function ReleasesPage() {
  return (
    <ProtectedRoute>
      <ReleasesContent />
    </ProtectedRoute>
  )
}

function ReleasesContent() {
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [registryFilter, setRegistryFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [classificationFilter, setClassificationFilter] = useState("")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useState<SortingState>([])

  const sort = sorting[0]
  const { data: releasesRes, isLoading } = useRecentReleases({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
    search: debouncedSearch || undefined,
    registry: registryFilter || undefined,
    status: statusFilter || undefined,
    classification: classificationFilter || undefined,
  })

  const releases = releasesRes?.data ?? []
  const total = releasesRes?.meta?.total ?? 0

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [sorting, debouncedSearch, registryFilter, statusFilter, classificationFilter])

  const columns = useMemo<ColumnDef<RecentRelease>[]>(
    () => [
      {
        accessorKey: "packageName",
        header: ({ column }) => <SortableHeader column={column} title="Package" />,
        cell: ({ row }) => (
          <Link
            href={`/releases/${row.original.id}`}
            className="font-medium hover:underline text-primary"
          >
            {row.original.packageName}
          </Link>
        ),
      },
      {
        accessorKey: "packageRegistry",
        header: ({ column }) => <SortableHeader column={column} title="Registry" />,
        cell: ({ row }) => (
          <Badge variant="outline">{row.original.packageRegistry}</Badge>
        ),
      },
      {
        accessorKey: "version",
        header: ({ column }) => <SortableHeader column={column} title="Version" />,
        cell: ({ row }) => (
          <span className="font-mono text-sm">{row.original.version}</span>
        ),
      },
      {
        accessorKey: "publishedAt",
        header: ({ column }) => <SortableHeader column={column} title="Published" />,
        cell: ({ row }) => (
          <span className="text-sm">
            {new Date(row.original.publishedAt).toLocaleDateString()}
          </span>
        ),
      },
      {
        accessorKey: "status",
        header: ({ column }) => <SortableHeader column={column} title="Status" />,
        cell: ({ row }) =>
          row.original.status === "completed" ? (
            <span className="flex items-center gap-1 text-sm text-green-600">
              <CheckCircle className="h-3.5 w-3.5" />
              Done
            </span>
          ) : (
            <Badge variant="secondary">{row.original.status}</Badge>
          ),
      },
      {
        accessorKey: "classification",
        header: "Classification",
        enableSorting: false,
        cell: ({ row }) =>
          row.original.classification ? (
            <Badge variant={classificationVariant(row.original.classification)}>
              {row.original.classification}
            </Badge>
          ) : (
            <span className="text-muted-foreground text-sm">—</span>
          ),
      },
    ],
    []
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  const table = useReactTable({
    data: releases,
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
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Releases</h1>
        <p className="text-sm text-muted-foreground">
          {total} total releases
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-4">
            <Input
              placeholder="Search packages..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="max-w-xs"
            />
            <Select
              value={registryFilter || "all"}
              onValueChange={(v) => setRegistryFilter(v === "all" ? "" : (v ?? ""))}
            >
              <SelectTrigger className="w-32">
                <SelectValue placeholder="All" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="pypi">PyPI</SelectItem>
                <SelectItem value="npm">npm</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={statusFilter || "all"}
              onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
            >
              <SelectTrigger className="w-36">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Statuses</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="diffing">Diffing</SelectItem>
                <SelectItem value="analyzing">Analyzing</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
                <SelectItem value="error">Error</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={classificationFilter || "all"}
              onValueChange={(v) => setClassificationFilter(v === "all" ? "" : (v ?? ""))}
            >
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All Classifications" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Classifications</SelectItem>
                <SelectItem value="benign">Benign</SelectItem>
                <SelectItem value="suspicious">Suspicious</SelectItem>
                <SelectItem value="malicious">Malicious</SelectItem>
                <SelectItem value="baseline">Baseline</SelectItem>
              </SelectContent>
            </Select>
            {(search || registryFilter || statusFilter || classificationFilter) && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSearch("")
                  setRegistryFilter("")
                  setStatusFilter("")
                  setClassificationFilter("")
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
              {isLoading && releases.length === 0 ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-12" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-14" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  </TableRow>
                ))
              ) : table.getRowModel().rows.length ? (
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
                    No releases yet. Add packages to start monitoring.
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
