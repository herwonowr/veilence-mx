"use client"

import { useEffect, useState, useMemo } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useDebouncedValue, useSortParams, useFilterParams, useResponsiveColumns, ROUTES, type ColumnBreakpoints , usePublicConfigQuery } from "@/core"
import Link from "next/link"
import { Card, CardContent, CardHeader, Badge, TableSkeleton, TableError, TableEmptyState, Button, Label, FilterChips, SearchInput, DataTablePagination, SortableHeader, ReleaseStatusBadge, ClassificationBadge, type SkeletonColumn, type ActiveFilter } from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import { formatVersion, type RecentRelease } from "@/domains/releases"
import type { Classification, Ecosystem, ReleaseStatus } from "@/domains/common"
import { Activity } from "lucide-react"
import { formatEcosystem } from "@/domains/common"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import { useRecentReleases } from "@/features/releases/hooks/use-releases"

const VALID_STATUSES: ReleaseStatus[] = ["in_progress", "completed", "error"]
const VALID_CLASSIFICATIONS: Classification[] = ["benign", "suspicious", "malicious", "baseline"]

export const ReleasesListView = () => {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { enabledEcosystems } = usePublicConfigQuery()
  const VALID_ECOSYSTEMS = enabledEcosystems as Ecosystem[]

  const initialEcosystem = searchParams.get("ecosystem") ?? ""
  const initialStatus = searchParams.get("status") ?? ""
  const initialClassification = searchParams.get("classification") ?? ""

  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [ecosystemFilter, setEcosystemFilter] = useState(
    VALID_ECOSYSTEMS.includes(initialEcosystem as Ecosystem) ? initialEcosystem : ""
  )
  const [statusFilter, setStatusFilter] = useState(
    VALID_STATUSES.includes(initialStatus as ReleaseStatus) ? initialStatus : ""
  )
  const [classificationFilter, setClassificationFilter] = useState(
    VALID_CLASSIFICATIONS.includes(initialClassification as Classification) ? initialClassification : ""
  )
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

  // Sync filter state → URL search params
  useFilterParams(
    useMemo(() => ({
      ecosystem: ecosystemFilter,
      status: statusFilter,
      classification: classificationFilter,
    }), [ecosystemFilter, statusFilter, classificationFilter]),
  )

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
      ? [{ label: "Ecosystem", value: formatEcosystem(ecosystemFilter), onRemove: () => setEcosystemFilter("") }]
      : []),
    ...(statusFilter
      ? [{ label: "Analysis Status", value: statusFilter === "in_progress" ? "In Progress" : statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1), onRemove: () => setStatusFilter("") }]
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
    { width: "w-16", header: "Analysis Status" },
    { width: "w-20", header: "Classification" },
  ]

  const columns = useMemo<ColumnDef<RecentRelease>[]>(
    () => [
      {
        accessorKey: "packageName",
        header: ({ column }) => <SortableHeader column={column} title="Package" />,
        cell: ({ row }) => (
          <Link
            href={ROUTES.RELEASE_DETAIL(row.original.id)}
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
          <span className="font-mono text-sm">{formatVersion(row.original.version)}</span>
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
        header: ({ column }) => <SortableHeader column={column} title="Analysis Status" />,
        cell: ({ row }) => <ReleaseStatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "classification",
        header: "Classification",
        enableSorting: false,
        cell: ({ row }) => <ClassificationBadge classification={row.original.classification} />,
      },
    ],
    []
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
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
                <Label htmlFor="releases-ecosystem-filter" className="text-xs text-muted-foreground">
                  Ecosystem
                </Label>
                <Select
                  value={ecosystemFilter || "all"}
                  onValueChange={(v) => setEcosystemFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="releases-ecosystem-filter" className="w-32">
                    <SelectValue>{ecosystemFilter ? formatEcosystem(ecosystemFilter) : "All"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    {VALID_ECOSYSTEMS.map((eco) => (
                      <SelectItem key={eco} value={eco}>{formatEcosystem(eco)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label htmlFor="releases-status-filter" className="text-xs text-muted-foreground">
                  Analysis Status
                </Label>
                <Select
                  value={statusFilter || "all"}
                  onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="releases-status-filter" className="w-36">
                    <SelectValue>{statusFilter === "in_progress" ? "In Progress" : statusFilter ? statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1) : "All Statuses"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Statuses</SelectItem>
                    <SelectItem value="in_progress">In Progress</SelectItem>
                    <SelectItem value="completed">Completed</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label htmlFor="releases-classification-filter" className="text-xs text-muted-foreground">
                  Classification
                </Label>
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
                    onClick={() => router.push(ROUTES.RELEASE_DETAIL(row.original.id))}
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
                    icon={<Activity className="h-8 w-8" />}
                    title="No matching releases."
                    description="Try adjusting your filters."
                  >
                    <Button variant="outline" size="sm" onClick={clearAllFilters}>
                      Clear filters
                    </Button>
                  </TableEmptyState>
                ) : (
                <TableEmptyState
                  colSpan={columns.length}
                  icon={<Activity className="h-8 w-8" />}
                  title="No releases yet."
                  description="Add packages to start monitoring releases."
                >
                  <Link href={ROUTES.PACKAGES}>
                    <Button variant="outline" size="sm">
                      Go to Packages
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
