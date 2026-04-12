"use client"

import { useEffect, useState, useMemo } from "react"
import { useRouter } from "next/navigation"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import Link from "next/link"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
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
import { TableSkeleton, type SkeletonColumn } from "@/components/table-skeleton"
import { TableError } from "@/components/table-error"
import { TableEmptyState } from "@/components/empty-state"
import { CheckCircle, Activity } from "lucide-react"
import { Button } from "@/components/ui/button"
import { FilterChips, type ActiveFilter } from "@/components/filter-chips"
import { SearchInput } from "@/components/search-input"
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
  const router = useRouter()
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

  const releaseColumnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    packageRegistry: "desktop",
    classification: "desktop",
    publishedAt: "tablet",
  }), [])
  const columnVisibility = useResponsiveColumns(releaseColumnBreakpoints)

  const sort = sorting[0]
  const { data: releasesRes, isLoading, isFetching, isError, refetch } = useRecentReleases({
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

  const hasActiveFilters = !!(search || registryFilter || statusFilter || classificationFilter)

  const clearAllFilters = () => {
    setSearch("")
    setRegistryFilter("")
    setStatusFilter("")
    setClassificationFilter("")
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(registryFilter
      ? [{ label: "Registry", value: registryFilter === "pypi" ? "PyPI" : "npm", onRemove: () => setRegistryFilter("") }]
      : []),
    ...(statusFilter
      ? [{ label: "Status", value: statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1), onRemove: () => setStatusFilter("") }]
      : []),
    ...(classificationFilter
      ? [{ label: "Classification", value: classificationFilter.charAt(0).toUpperCase() + classificationFilter.slice(1), onRemove: () => setClassificationFilter("") }]
      : []),
    ...(search
      ? [{ label: "Search", value: search, onRemove: () => setSearch("") }]
      : []),
  ]

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-24", header: "Package" },
    { width: "w-16", header: "Registry" },
    { width: "w-20", header: "Version" },
    { width: "w-24", header: "Published" },
    { width: "w-16", header: "Status" },
    { width: "w-20", header: "Classification" },
  ]

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
    state: { pagination, sorting, columnVisibility },
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
          <div className="space-y-3">
            <div className="flex flex-wrap items-end gap-4">
              <SearchInput
                value={search}
                onChange={setSearch}
                onClear={() => setSearch("")}
                isLoading={isFetching && !!debouncedSearch}
                placeholder="Search packages..."
                aria-label="Search releases"
              />
              <div className="space-y-1">
                <label htmlFor="releases-registry-filter" className="text-xs font-medium text-muted-foreground">
                  Registry
                </label>
                <Select
                  value={registryFilter || "all"}
                  onValueChange={(v) => setRegistryFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="releases-registry-filter" className="w-32">
                    <SelectValue placeholder="All" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="pypi">PyPI</SelectItem>
                    <SelectItem value="npm">npm</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <label htmlFor="releases-status-filter" className="text-xs font-medium text-muted-foreground">
                  Status
                </label>
                <Select
                  value={statusFilter || "all"}
                  onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="releases-status-filter" className="w-36">
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
              </div>
              <div className="space-y-1">
                <label htmlFor="releases-classification-filter" className="text-xs font-medium text-muted-foreground">
                  Classification
                </label>
                <Select
                  value={classificationFilter || "all"}
                  onValueChange={(v) => setClassificationFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="releases-classification-filter" className="w-40">
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
                    onClick={() => router.push(`/releases/${row.original.id}`)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableEmptyState
                  colSpan={columns.length}
                  icon={<Activity className="h-8 w-8" />}
                  title="No releases yet."
                  description="Add packages to start monitoring releases."
                >
                  <Link href="/packages">
                    <Button variant="outline" size="sm">
                      Go to Packages
                    </Button>
                  </Link>
                </TableEmptyState>
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
