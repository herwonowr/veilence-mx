"use client"

import { useState, useMemo, useEffect } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Link from "next/link"
import {
  useDebouncedValue,
  useSortParams,
  useFilterParams,
  useResponsiveColumns,
  ROUTES,
  usePublicConfigQuery,
  type ColumnBreakpoints,
} from "@/core"
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  FilterChips,
  SearchInput,
  Label,
  DataTablePagination,
  SortableHeader,
  TableSkeleton,
  TableError,
  TableEmptyState,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/ui"
import { formatEcosystem } from "@/domains/common"
import type { Ecosystem } from "@/domains/common"
import { formatPopularity, popularityTooltip } from "@/domains/packages"
import type { StalePackage } from "@/domains/packages"
import { formatVersion } from "@/domains/releases"
import { ArrowLeft, Clock } from "lucide-react"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import { useStalePackages } from "@/features/packages/hooks/use-packages"

const THRESHOLD_OPTIONS = [1, 3, 6, 12, 18, 24]
const DEFAULT_MONTHS = 6

export const StalePackagesView = () => {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { enabledEcosystems } = usePublicConfigQuery()
  const VALID_ECOSYSTEMS = enabledEcosystems as Ecosystem[]

  const initialEcosystem = searchParams.get("ecosystem") ?? ""
  const initialMonths = parseInt(searchParams.get("months") ?? "", 10)
  const initialSearch = searchParams.get("search") ?? ""

  const [months, setMonths] = useState(
    THRESHOLD_OPTIONS.includes(initialMonths) ? initialMonths : DEFAULT_MONTHS
  )
  const [ecosystemFilter, setEcosystemFilter] = useState(
    VALID_ECOSYSTEMS.includes(initialEcosystem as Ecosystem) ? initialEcosystem : ""
  )
  const [search, setSearch] = useState(initialSearch)
  const debouncedSearch = useDebouncedValue(search, 300)
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

  const STALE_FILTER_DEFAULTS = useMemo(() => ({ months: String(DEFAULT_MONTHS) }), [])
  useFilterParams(
    useMemo(() => ({
      ecosystem: ecosystemFilter,
      months: String(months),
      search,
    }), [ecosystemFilter, months, search]),
    STALE_FILTER_DEFAULTS,
  )

  const columnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    downloadCount: "desktop",
    latestVersion: "tablet",
  }), [])
  const columnVisibility = useResponsiveColumns(columnBreakpoints)

  // Reset to first page when filters or sort change
  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [ecosystemFilter, debouncedSearch, sorting, months])

  const sort = sorting[0]
  const {
    data: staleRes,
    isLoading,
    isFetching,
    isError,
    refetch,
  } = useStalePackages({
    months,
    ecosystem: ecosystemFilter || undefined,
    search: debouncedSearch || undefined,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const stalePackages = staleRes?.data ?? []
  const total = staleRes?.meta?.total ?? 0

  const hasActiveFilters = !!(search || ecosystemFilter || months !== DEFAULT_MONTHS)

  const clearAllFilters = () => {
    setSearch("")
    setEcosystemFilter("")
    setMonths(DEFAULT_MONTHS)
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(ecosystemFilter
      ? [{ label: "Ecosystem", value: formatEcosystem(ecosystemFilter), onRemove: () => setEcosystemFilter("") }]
      : []),
    ...(months !== DEFAULT_MONTHS
      ? [{ label: "Threshold", value: `${months} month${months !== 1 ? "s" : ""}`, onRemove: () => setMonths(DEFAULT_MONTHS) }]
      : []),
    ...(search
      ? [{ label: "Search", value: search, onRemove: () => setSearch("") }]
      : []),
  ]

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Name" },
    { width: "w-16", header: "Ecosystem" },
    { width: "w-20", header: "Latest Version" },
    { width: "w-24", header: "Last Release" },
    { width: "w-16", header: "Days Since" },
    { width: "w-20", header: "Popularity" },
  ]

  const columns = useMemo<ColumnDef<StalePackage>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <SortableHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <Link
            href={ROUTES.PACKAGE_DETAIL(row.original.id)}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ),
      },
      {
        accessorKey: "ecosystem",
        header: ({ column }) => <SortableHeader column={column} title="Ecosystem" />,
        cell: ({ row }) => (
          <Badge variant="outline">{formatEcosystem(row.original.ecosystem)}</Badge>
        ),
      },
      {
        id: "latestVersion",
        accessorKey: "latestVersion",
        header: ({ column }) => <SortableHeader column={column} title="Latest Version" />,
        cell: ({ row }) => row.original.latestVersion ? formatVersion(row.original.latestVersion) : "-",
      },
      {
        id: "lastRelease",
        header: "Last Release",
        enableSorting: false,
        cell: ({ row }) =>
          row.original.lastReleaseAt
            ? new Date(row.original.lastReleaseAt).toLocaleDateString()
            : "Never",
      },
      {
        accessorKey: "daysSinceLastRelease",
        header: ({ column }) => <SortableHeader column={column} title="Days Since" />,
        cell: ({ row }) => {
          const days = row.original.daysSinceLastRelease
          return (
            <span className={days > 365 ? "text-destructive font-medium" : ""}>
              {days}
            </span>
          )
        },
      },
      {
        id: "downloadCount",
        accessorKey: "downloadCount",
        header: ({ column }) => (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger render={<span />}>
                <SortableHeader column={column} title="Popularity" />
              </TooltipTrigger>
              <TooltipContent className="whitespace-pre-line">
                {"NPM & Python: Monthly downloads\nGolang: GitHub stars"}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ),
        cell: ({ row }) => {
          const pkg = row.original
          const text = formatPopularity(pkg.ecosystem, pkg.downloadCount)
          const tooltip = popularityTooltip(pkg.ecosystem, pkg.downloadCount, pkg.downloadCountUpdatedAt)
          return (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger render={<span className="text-sm tabular-nums">{text}</span>} />
                <TooltipContent>{tooltip}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )
        },
      },
    ],
    []
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
  const table = useReactTable({
    data: stalePackages,
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
      <div>
        <Link
          href={ROUTES.PACKAGES}
          className="w-fit text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Packages
        </Link>
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Stale Packages</h1>
          <p className="text-sm text-muted-foreground">
            {total} total stale packages
          </p>
        </div>
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
                placeholder="Search stale packages..."
                aria-label="Search stale packages"
              />
              <div className="space-y-1">
                <Label htmlFor="stale-ecosystem-filter" className="text-xs text-muted-foreground">
                  Ecosystem
                </Label>
                <Select
                  value={ecosystemFilter || "all"}
                  onValueChange={(v) => setEcosystemFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="stale-ecosystem-filter" className="w-32">
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
                <Label htmlFor="stale-threshold" className="text-xs text-muted-foreground">
                  No releases in
                </Label>
                <Select
                  value={String(months)}
                  onValueChange={(v) => { if (v) setMonths(parseInt(v, 10)) }}
                >
                  <SelectTrigger id="stale-threshold" className="w-32">
                    <SelectValue>{months} month{months !== 1 ? "s" : ""}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {THRESHOLD_OPTIONS.map((m) => (
                      <SelectItem key={m} value={String(m)}>
                        {m} month{m !== 1 ? "s" : ""}
                      </SelectItem>
                    ))}
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
                            style={{ width: header.getSize() !== 150 ? header.getSize() : undefined }}
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
                        onClick={() => router.push(ROUTES.PACKAGE_DETAIL(row.original.id))}
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
                        icon={<Clock className="h-8 w-8" />}
                        title="No matching stale packages."
                        description="Try adjusting your filters."
                      >
                        <Button variant="outline" size="sm" onClick={clearAllFilters}>
                          Clear filters
                        </Button>
                      </TableEmptyState>
                    ) : (
                      <TableEmptyState
                        colSpan={columns.length}
                        icon={<Clock className="h-8 w-8" />}
                        title="No stale packages."
                        description={`All monitored packages have had a release within the last ${months} months.`}
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
    </div>
  )
}
