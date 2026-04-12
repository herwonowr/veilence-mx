"use client"

import { useEffect, useState, useMemo } from "react"
import { useRouter } from "next/navigation"
import { useDebouncedValue } from "@/core/hooks/use-debounced-value"
import { useSortParams } from "@/core/hooks/use-sort-params"
import Link from "next/link"
import { Card, CardContent, CardHeader } from "@/ui/components/card"
import { Badge } from "@/ui/components/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui/components/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import type { RecentRelease } from "@/domains/releases"
import type { Classification } from "@/domains/common"
import { TableSkeleton, type SkeletonColumn } from "@/ui/feedback/table-skeleton"
import { TableError } from "@/ui/feedback/table-error"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { CheckCircle, Activity } from "lucide-react"
import { Button } from "@/ui/components/button"
import { FilterChips, type ActiveFilter } from "@/ui/data/filter-chips"
import { SearchInput } from "@/ui/form/search-input"
import { formatEcosystem } from "@/domains/common"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import { DataTablePagination } from "@/ui/data/data-table-pagination"
import { SortableHeader } from "@/ui/data/sortable-header"
import { useResponsiveColumns, type ColumnBreakpoints } from "@/core/hooks/use-responsive-columns"
import { useRecentReleases } from "@/features/releases/hooks/use-releases"

const classificationVariant = (c: Classification) => {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  if (c === "baseline") return "outline" as const
  return "secondary" as const
}

export const ReleasesListView = () => {
  const router = useRouter()
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [ecosystemFilter, setEcosystemFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [classificationFilter, setClassificationFilter] = useState("")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

  const releaseColumnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    publishedAt: "desktop",
    packageEcosystem: "tablet",
  }), [])
  const columnVisibility = useResponsiveColumns(releaseColumnBreakpoints)

  const sort = sorting[0]
  const { data: releasesRes, isLoading, isFetching, isError, refetch } = useRecentReleases({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
    search: debouncedSearch || undefined,
    ecosystem: ecosystemFilter || undefined,
    status: statusFilter || undefined,
    classification: classificationFilter || undefined,
  })

  const releases = releasesRes?.data ?? []
  const total = releasesRes?.meta?.total ?? 0

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [sorting, debouncedSearch, ecosystemFilter, statusFilter, classificationFilter])

  const hasActiveFilters = !!(search || ecosystemFilter || statusFilter || classificationFilter)

  const clearAllFilters = () => {
    setSearch("")
    setEcosystemFilter("")
    setStatusFilter("")
    setClassificationFilter("")
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(ecosystemFilter
      ? [{ label: "Ecosystem", value: ecosystemFilter === "python" ? "Python" : "NPM", onRemove: () => setEcosystemFilter("") }]
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
    { width: "w-16", header: "Ecosystem" },
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
        accessorKey: "packageEcosystem",
        header: ({ column }) => <SortableHeader column={column} title="Ecosystem" />,
        cell: ({ row }) => (
          <Badge variant="outline">{formatEcosystem(row.original.packageEcosystem)}</Badge>
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
                <label htmlFor="releases-ecosystem-filter" className="text-xs font-medium text-muted-foreground">
                  Ecosystem
                </label>
                <Select
                  value={ecosystemFilter || "all"}
                  onValueChange={(v) => setEcosystemFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="releases-ecosystem-filter" className="w-32">
                    <SelectValue>{ecosystemFilter === "python" ? "Python" : ecosystemFilter === "npm" ? "NPM" : "All"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="python">Python</SelectItem>
                    <SelectItem value="npm">NPM</SelectItem>
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
                    <SelectValue>{statusFilter ? statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1) : "All Statuses"}</SelectValue>
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
                    <SelectValue>{classificationFilter ? classificationFilter.charAt(0).toUpperCase() + classificationFilter.slice(1) : "All Classifications"}</SelectValue>
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
