"use client"

import { useEffect, useState, useMemo, useCallback } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useDebouncedValue, useSortParams, useFilterParams, useResponsiveColumns, useCurrentWorkspaceRole, hasMinimumRole, ROUTES, type ColumnBreakpoints , usePublicConfigQuery, parseFieldErrors } from "@/core"
import { Button, Badge, Input, Textarea, Card, CardContent, CardHeader, TableSkeleton, TableError, TableEmptyState, FilterChips, SearchInput, Field, FieldLabel, FieldError, Label, DataTablePagination, SortableHeader, type SkeletonColumn, type ActiveFilter } from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/ui"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import type { Package, PackageSource, PackageStatus } from "@/domains/packages"
import type { Ecosystem } from "@/domains/common"
import { formatPopularity, popularityTooltip, packageSchema } from "@/domains/packages"
import { Plus, Trash2, RefreshCw, Upload, Ban, ShieldCheck, Radar, Clock } from "lucide-react"
import { formatEcosystem } from "@/domains/common"
import { formatVersion } from "@/domains/releases"
import Link from "next/link"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/ui"
import {
  usePackages,
  useCreatePackage,
  useDeletePackage,
  useBlockPackage,
  useUnblockPackage,
  useDiscoverPackages,
  useSuggestionCount,
} from "@/features/packages/hooks/use-packages"
import { PipelineStatusBar } from "@/features/packages/ui/pipeline-status-bar"

const formatSource = (source: PackageSource): string => {
  switch (source) {
    case "manual":
      return "Manual"
    case "discovered":
      return "Discovered"
    case "imported":
      return "Imported"
    default:
      return source
  }
}

const sourceVariant = (source: PackageSource): "default" | "secondary" | "outline" => {
  switch (source) {
    case "manual":
      return "default"
    case "discovered":
      return "secondary"
    case "imported":
      return "outline"
    default:
      return "secondary"
  }
}

const VALID_STATUSES: PackageStatus[] = ["active", "suggested", "blocked", "removed"]
const VALID_SOURCES: PackageSource[] = ["manual", "discovered", "imported"]

export const PackagesListView = () => {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { enabledEcosystems } = usePublicConfigQuery()
  const VALID_ECOSYSTEMS = enabledEcosystems as Ecosystem[]

  const initialEcosystem = searchParams.get("ecosystem") ?? ""
  const initialStatus = searchParams.get("status") ?? ""
  const initialSource = searchParams.get("source") ?? ""

  const [ecosystemFilter, setEcosystemFilter] = useState(
    VALID_ECOSYSTEMS.includes(initialEcosystem as Ecosystem) ? initialEcosystem : ""
  )
  const [statusFilter, setStatusFilter] = useState<string>(
    VALID_STATUSES.includes(initialStatus as PackageStatus) ? initialStatus : ""
  )
  const [sourceFilter, setSourceFilter] = useState<string>(
    VALID_SOURCES.includes(initialSource as PackageSource) ? initialSource : ""
  )
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [newName, setNewName] = useState("")
  const [newEcosystem, setNewEcosystem] = useState<Ecosystem>("python")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()
  const [removeTarget, setRemoveTarget] = useState<{ id: string; name: string } | null>(null)

  // Sync filter state → URL search params
  // For packages, "" (all) is the default status - omit from URL when it matches
  const PACKAGES_FILTER_DEFAULTS = useMemo(() => ({ status: "" }), [])
  useFilterParams(
    useMemo(() => ({
      ecosystem: ecosystemFilter,
      status: statusFilter,
      source: sourceFilter,
    }), [ecosystemFilter, statusFilter, sourceFilter]),
    PACKAGES_FILTER_DEFAULTS,
  )
  const [blockTarget, setBlockTarget] = useState<{ id: string; name: string } | null>(null)
  const [blockReason, setBlockReason] = useState("")
  const [unblockTarget, setUnblockTarget] = useState<{ id: string; name: string } | null>(null)
  const [createErrors, setCreateErrors] = useState<Record<string, string>>({})
  const [createFormSubmitted, setCreateFormSubmitted] = useState(false)

  const validateCreate = (fields: { name: string; ecosystem: string }) => {
    if (!createFormSubmitted) return
    const result = packageSchema.safeParse(fields)
    setCreateErrors(result.success ? {} : parseFieldErrors(result.error))
  }
  const packageColumnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    downloadCount: "desktop",
    source: "tablet",
    status: "tablet",
  }), [])
  const columnVisibility = useResponsiveColumns(packageColumnBreakpoints)

  const sort = sorting[0]
  const { data: packagesRes, isLoading, isFetching, isError, refetch } = usePackages({
    ecosystem: ecosystemFilter || undefined,
    search: debouncedSearch || undefined,
    status: statusFilter || undefined,
    source: sourceFilter || undefined,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const packages = packagesRes?.data ?? []
  const total = packagesRes?.meta?.total ?? 0

  const createMutation = useCreatePackage()
  const deleteMutation = useDeletePackage()
  const blockMutation = useBlockPackage()
  const unblockMutation = useUnblockPackage()
  const discoverMutation = useDiscoverPackages()

  const { data: suggestionsCount = 0 } = useSuggestionCount()

  const { role } = useCurrentWorkspaceRole()
  /** Viewers get a read-only view - all mutation buttons are hidden */
  const canWrite = hasMinimumRole(role, "member")
  /** Package deletion requires admin+ (packages:delete permission) */
  const canDelete = hasMinimumRole(role, "admin")

  // Reset to first page when filters or sort change
  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [ecosystemFilter, statusFilter, sourceFilter, debouncedSearch, sorting])

  const hasActiveFilters = !!(search || ecosystemFilter || sourceFilter || statusFilter)

  const clearAllFilters = () => {
    setSearch("")
    setEcosystemFilter("")
    setStatusFilter("")
    setSourceFilter("")
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(ecosystemFilter
      ? [{ label: "Ecosystem", value: formatEcosystem(ecosystemFilter), onRemove: () => setEcosystemFilter("") }]
      : []),
    ...(statusFilter
      ? [{ label: "Status", value: statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1), onRemove: () => setStatusFilter("") }]
      : []),
    ...(sourceFilter
      ? [{ label: "Source", value: formatSource(sourceFilter as PackageSource), onRemove: () => setSourceFilter("") }]
      : []),
    ...(search
      ? [{ label: "Search", value: search, onRemove: () => setSearch("") }]
      : []),
  ]

  const handleCreate = async () => {
    setCreateFormSubmitted(true)
    setCreateErrors({})
    try {
      const data = packageSchema.parse({ name: newName, ecosystem: newEcosystem })
      await createMutation.mutateAsync({ name: data.name, ecosystem: data.ecosystem })
      setDialogOpen(false)
      setCreateFormSubmitted(false)
      setNewName("")
    } catch (err) {
      setCreateErrors(parseFieldErrors(err))
    }
  }

  const handleRemove = useCallback(
    (id: string, name: string) => {
      setRemoveTarget({ id, name })
    },
    []
  )

  const confirmRemove = useCallback(() => {
    if (removeTarget) {
      deleteMutation.mutate(removeTarget.id)
      setRemoveTarget(null)
    }
  }, [removeTarget, deleteMutation])

  const handleBlock = useCallback(
    (id: string, name: string) => {
      setBlockTarget({ id, name })
      setBlockReason("")
    },
    []
  )

  const confirmBlock = useCallback(() => {
    if (blockTarget) {
      blockMutation.mutate({ id: blockTarget.id, reason: blockReason || undefined })
      setBlockTarget(null)
      setBlockReason("")
    }
  }, [blockTarget, blockReason, blockMutation])

  const handleUnblock = useCallback(
    (id: string, name: string) => {
      setUnblockTarget({ id, name })
    },
    []
  )

  const confirmUnblock = useCallback(() => {
    if (unblockTarget) {
      unblockMutation.mutate(unblockTarget.id)
      setUnblockTarget(null)
    }
  }, [unblockTarget, unblockMutation])

  const handleDiscover = () => {
    discoverMutation.mutate(undefined)
  }

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Name" },
    { width: "w-16", header: "Ecosystem" },
    { width: "w-20", header: "Latest Version" },
    { width: "w-16", header: "Popularity" },
    { width: "w-16", header: "Source" },
    { width: "w-16", header: "Status" },
    { width: "w-8", header: "" },
  ]

  const columns = useMemo<ColumnDef<Package>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <SortableHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <Link
            href={ROUTES.PACKAGE_DETAIL(row.original.id)}
            className={`font-medium hover:underline ${row.original.status === "blocked" ? "text-muted-foreground" : ""}`}
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
        accessorKey: "latestVersion",
        header: ({ column }) => <SortableHeader column={column} title="Latest Version" />,
        cell: ({ row }) => row.original.latestVersion ? formatVersion(row.original.latestVersion) : "-",
      },
      {
        id: "downloadCount",
        accessorKey: "downloadCount",
        header: ({ column }) => {
          return (
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
          )
        },
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
      {
        accessorKey: "source",
        header: "Source",
        enableSorting: false,
        cell: ({ row }) => (
          <Badge variant={sourceVariant(row.original.source)}>
            {formatSource(row.original.source)}
          </Badge>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        enableSorting: false,
        cell: ({ row }) => {
          if (row.original.status === "blocked") {
            return (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Badge variant="destructive">
                        <Ban className="h-3 w-3 mr-1" />
                        Blocked
                      </Badge>
                    }
                  />
                  {row.original.blockedReason && (
                    <TooltipContent>{row.original.blockedReason}</TooltipContent>
                  )}
                </Tooltip>
              </TooltipProvider>
            )
          }
          if (row.original.status === "suggested") {
            return (
              <Badge variant="outline">Suggested</Badge>
            )
          }
          return (
            <Badge variant="secondary">
              <ShieldCheck className="h-3 w-3 mr-1" />
              Active
            </Badge>
          )
        },
      },
      {
        id: "actions",
        header: () => <span className="sr-only">Actions</span>,
        enableSorting: false,
        meta: { headerClassName: "w-[1%] whitespace-nowrap text-right", cellClassName: "text-right" },
        cell: ({ row }) => {
          if (!canWrite) return null
          const pkg = row.original
          if (pkg.status === "blocked") {
            return (
              <div className="flex items-center justify-end gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Unblock package ${pkg.name}`}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleUnblock(pkg.id, pkg.name)
                  }}
                >
                  <ShieldCheck className="size-3.5 text-emerald-600 dark:text-emerald-500 mr-1" />
                  Unblock
                </Button>
              </div>
            )
          }
          return (
            <div className="flex items-center justify-end gap-1">
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={`Block package ${pkg.name}`}
                onClick={(e) => {
                  e.stopPropagation()
                  handleBlock(pkg.id, pkg.name)
                }}
              >
                <Ban className="size-4 text-destructive" />
              </Button>
              {canDelete && (
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={`Remove package ${pkg.name}`}
                onClick={(e) => {
                  e.stopPropagation()
                  handleRemove(pkg.id, pkg.name)
                }}
              >
                <Trash2 className="size-4 text-destructive" />
              </Button>
              )}
            </div>
          )
        },
      },
    ],
    [handleRemove, handleBlock, handleUnblock, canWrite, canDelete]
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
  const table = useReactTable({
    data: packages,
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
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-3xl font-bold">Packages</h1>
          <Link href={ROUTES.PACKAGES_STALE}>
            <Button variant="outline" size="sm" className="gap-1.5">
              <Clock className="h-4 w-4" />
              Stale Packages
            </Button>
          </Link>
          <Link href={ROUTES.PACKAGES_SUGGESTIONS}>
            <Button variant="outline" size="sm" className="gap-1.5">
              <Radar className="h-4 w-4" />
              Suggestions
              {suggestionsCount > 0 && (
                <Badge variant="secondary" className="ml-0.5 px-1.5 py-0 text-xs min-w-5 justify-center">
                  {suggestionsCount}
                </Badge>
              )}
            </Button>
          </Link>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {canWrite && (
            <>
              <Button variant="outline" onClick={handleDiscover} disabled={discoverMutation.isPending}>
                <RefreshCw className={`h-4 w-4 mr-2 ${discoverMutation.isPending ? "animate-spin" : ""}`} />
                Discover Packages
              </Button>
              <Link href={ROUTES.PACKAGES_IMPORT}>
                <Button variant="outline">
                  <Upload className="h-4 w-4 mr-2" />
                  Bulk Import
                </Button>
              </Link>
              <Dialog open={dialogOpen} onOpenChange={(open) => setDialogOpen(open)}>
                <DialogTrigger
                  render={
                    <Button>
                      <Plus className="h-4 w-4 mr-2" />
                      Add Package
                    </Button>
                  }
                />
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Add Custom Package</DialogTitle>
                  </DialogHeader>
                  <div className="space-y-4 pt-4">
                    <Field data-invalid={!!createErrors.name}>
                      <FieldLabel htmlFor="package-name">
                        Package Name
                      </FieldLabel>
                      <Input
                        id="package-name"
                        value={newName}
                        onChange={(e) => { setNewName(e.target.value); validateCreate({ name: e.target.value, ecosystem: newEcosystem }) }}
                        placeholder="e.g., requests"
                      />
                      {createErrors.name && (
                        <FieldError>{createErrors.name}</FieldError>
                      )}
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="package-ecosystem">
                        Ecosystem
                      </FieldLabel>
                      <Select
                        value={newEcosystem}
                        onValueChange={(v) => { if (v) { setNewEcosystem(v as Ecosystem); validateCreate({ name: newName, ecosystem: v }) } }}
                      >
                        <SelectTrigger id="package-ecosystem">
                          <SelectValue>{formatEcosystem(newEcosystem)}</SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          {VALID_ECOSYSTEMS.map((eco) => (
                            <SelectItem key={eco} value={eco}>{formatEcosystem(eco)}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </Field>
                    <Button onClick={handleCreate} className="w-full" disabled={createMutation.isPending}>
                      Add Package
                    </Button>
                  </div>
                </DialogContent>
              </Dialog>
            </>
          )}
          {!canWrite && role === "viewer" && (
            <Badge variant="outline" className="text-muted-foreground">
              Read-only
            </Badge>
          )}
        </div>
      </div>

      {suggestionsCount > 0 && (
        <Link
          href={ROUTES.PACKAGES_SUGGESTIONS}
          className="flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-800 hover:bg-amber-100 transition-colors dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300 dark:hover:bg-amber-900"
        >
          <Radar className="h-4 w-4 shrink-0" />
          <span>
            <strong>{suggestionsCount}</strong> suggestion{suggestionsCount !== 1 ? "s" : ""} pending review
          </span>
        </Link>
      )}

      <PipelineStatusBar />

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
                aria-label="Search packages"
              />
              <div className="space-y-1">
                <Label htmlFor="packages-ecosystem-filter" className="text-xs text-muted-foreground">
                  Ecosystem
                </Label>
                <Select
                  value={ecosystemFilter || "all"}
                  onValueChange={(v) => setEcosystemFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="packages-ecosystem-filter" className="w-32">
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
                <Label htmlFor="packages-status-filter" className="text-xs text-muted-foreground">
                  Status
                </Label>
                <Select
                  value={statusFilter || "all"}
                  onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="packages-status-filter" className="w-32">
                    <SelectValue>
                      {statusFilter === "active" ? "Active" : statusFilter === "blocked" ? "Blocked" : statusFilter === "suggested" ? "Suggested" : "All"}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="suggested">Suggested</SelectItem>
                    <SelectItem value="blocked">Blocked</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label htmlFor="packages-source-filter" className="text-xs text-muted-foreground">
                  Source
                </Label>
                <Select
                  value={sourceFilter || "all"}
                  onValueChange={(v) => setSourceFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="packages-source-filter" className="w-36">
                    <SelectValue>
                      {sourceFilter ? formatSource(sourceFilter as PackageSource) : "All"}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="manual">Manual</SelectItem>
                    <SelectItem value="discovered">Discovered</SelectItem>
                    <SelectItem value="imported">Imported</SelectItem>
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
                    className={row.original.status === "blocked" ? "opacity-50" : undefined}
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
                    icon={<Plus className="h-8 w-8" />}
                    title="No matching packages."
                    description="Try adjusting your filters."
                  >
                    <Button variant="outline" size="sm" onClick={clearAllFilters}>
                      Clear filters
                    </Button>
                  </TableEmptyState>
                ) : (
                <TableEmptyState
                  colSpan={columns.length}
                  icon={<Plus className="h-8 w-8" />}
                  title="No packages found."
                  description={canWrite ? "Add your first package or discover popular packages to start monitoring." : "No packages are being monitored yet."}
                >
                  {canWrite && (
                    <>
                      <Button size="sm" onClick={() => setDialogOpen(true)}>
                        Add Package
                      </Button>
                      <Button variant="outline" size="sm" onClick={handleDiscover} disabled={discoverMutation.isPending}>
                        Discover Packages
                      </Button>
                    </>
                  )}
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

      {/* Remove Confirmation Dialog */}
      <AlertDialog open={!!removeTarget} onOpenChange={(open) => { if (!open) setRemoveTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Package</AlertDialogTitle>
            <AlertDialogDescription>
              Remove <strong>{removeTarget?.name}</strong> from monitoring? If this is a popular package,
              automatic discovery may re-add it. Use <strong>Block</strong> to permanently exclude a package
              from discovery and analysis.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmRemove}>
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Block Confirmation Dialog */}
      <AlertDialog open={!!blockTarget} onOpenChange={(open) => { if (!open) { setBlockTarget(null); setBlockReason("") } }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Block Package</AlertDialogTitle>
            <AlertDialogDescription>
              Block <strong>{blockTarget?.name}</strong>? This permanently excludes the package from
              automatic discovery and analysis. You can unblock it later.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="px-6 pb-2">
            <Field>
              <FieldLabel htmlFor="block-reason">
                Reason (optional)
              </FieldLabel>
              <Textarea
                id="block-reason"
                value={blockReason}
                onChange={(e) => setBlockReason(e.target.value)}
                placeholder="e.g., Known benign package with noisy diffs"
                rows={2}
              />
            </Field>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmBlock}>
              Block
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Unblock Confirmation Dialog */}
      <AlertDialog open={!!unblockTarget} onOpenChange={(open) => { if (!open) setUnblockTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unblock Package</AlertDialogTitle>
            <AlertDialogDescription>
              Unblock <strong>{unblockTarget?.name}</strong>? This will re-enable automatic discovery
              and analysis for this package.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmUnblock}>
              Unblock
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
