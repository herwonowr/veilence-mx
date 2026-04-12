"use client"

import { useEffect, useState, useMemo, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useDebouncedValue } from "@/hooks/use-debounced-value"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { Package, Registry } from "@/types"
import { Plus, Trash2, RefreshCw, Upload } from "lucide-react"
import { TableSkeleton, type SkeletonColumn } from "@/components/table-skeleton"
import { TableError } from "@/components/table-error"
import { TableEmptyState } from "@/components/empty-state"
import { FilterChips, type ActiveFilter } from "@/components/filter-chips"
import { SearchInput } from "@/components/search-input"
import Link from "next/link"
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
import { packageSchema } from "@/lib/validations"
import { ZodError } from "zod"
import {
  usePackages,
  useCreatePackage,
  useDeletePackage,
  useSyncTopPackages,
} from "@/features/packages"

export default function PackagesPage() {
  return (
    <ProtectedRoute>
      <PackagesContent />
    </ProtectedRoute>
  )
}

function PackagesContent() {
  const router = useRouter()
  const [registryFilter, setRegistryFilter] = useState("")
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [newName, setNewName] = useState("")
  const [newRegistry, setNewRegistry] = useState<Registry>("pypi")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useState<SortingState>([])
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; name: string } | null>(null)
  const [createErrors, setCreateErrors] = useState<Record<string, string>>({})

  const packageColumnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    rank: "desktop",
    isCustom: "desktop",
    latestVersion: "tablet",
  }), [])
  const columnVisibility = useResponsiveColumns(packageColumnBreakpoints)

  const sort = sorting[0]
  const { data: packagesRes, isLoading, isFetching, isError, refetch } = usePackages({
    registry: registryFilter || undefined,
    search: debouncedSearch || undefined,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const packages = packagesRes?.data ?? []
  const total = packagesRes?.meta?.total ?? 0

  const createMutation = useCreatePackage()
  const deleteMutation = useDeletePackage()
  const syncMutation = useSyncTopPackages()

  // Reset to first page when filters or sort change
  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [registryFilter, debouncedSearch, sorting])

  const hasActiveFilters = !!(search || registryFilter)

  const clearAllFilters = () => {
    setSearch("")
    setRegistryFilter("")
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(registryFilter
      ? [{ label: "Registry", value: registryFilter === "pypi" ? "PyPI" : "npm", onRemove: () => setRegistryFilter("") }]
      : []),
    ...(search
      ? [{ label: "Search", value: search, onRemove: () => setSearch("") }]
      : []),
  ]

  const handleCreate = async () => {
    setCreateErrors({})
    try {
      const data = packageSchema.parse({ name: newName, registry: newRegistry })
      await createMutation.mutateAsync({ name: data.name, registry: data.registry })
      setDialogOpen(false)
      setNewName("")
    } catch (err) {
      if (err instanceof ZodError) {
        const fieldErrors: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") fieldErrors[key] = issue.message
        }
        setCreateErrors(fieldErrors)
      }
    }
  }

  const handleDelete = useCallback(
    (id: number, name: string) => {
      setDeleteTarget({ id, name })
    },
    []
  )

  const confirmDelete = useCallback(() => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id)
      setDeleteTarget(null)
    }
  }, [deleteTarget, deleteMutation])

  const handleSync = () => {
    syncMutation.mutate(undefined)
  }

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Name" },
    { width: "w-16", header: "Registry" },
    { width: "w-20", header: "Latest Version" },
    { width: "w-12", header: "Rank" },
    { width: "w-16", header: "Type" },
    { width: "w-8", header: "" },
  ]

  const columns = useMemo<ColumnDef<Package>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <SortableHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <Link
            href={`/packages/${row.original.id}`}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ),
      },
      {
        accessorKey: "registry",
        header: ({ column }) => <SortableHeader column={column} title="Registry" />,
        cell: ({ row }) => (
          <Badge variant="outline">{row.original.registry}</Badge>
        ),
      },
      {
        accessorKey: "latestVersion",
        header: ({ column }) => <SortableHeader column={column} title="Latest Version" />,
        cell: ({ row }) => row.original.latestVersion || "—",
      },
      {
        accessorKey: "rank",
        header: ({ column }) => <SortableHeader column={column} title="Rank" />,
        cell: ({ row }) => row.original.rank ?? "—",
      },
      {
        accessorKey: "isCustom",
        header: "Type",
        enableSorting: false,
        cell: ({ row }) => (
          <Badge variant={row.original.isCustom ? "default" : "secondary"}>
            {row.original.isCustom ? "Custom" : "Top-N"}
          </Badge>
        ),
      },
      {
        id: "actions",
        header: "",
        enableSorting: false,
        size: 48,
        cell: ({ row }) => (
          <Button
            variant="ghost"
            size="icon"
            aria-label={`Delete package ${row.original.name}`}
            onClick={() => handleDelete(row.original.id, row.original.name)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        ),
      },
    ],
    [handleDelete]
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

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
        <h1 className="text-3xl font-bold">Packages</h1>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" onClick={handleSync} disabled={syncMutation.isPending}>
            <RefreshCw className={`h-4 w-4 mr-2 ${syncMutation.isPending ? "animate-spin" : ""}`} />
            Sync Top Packages
          </Button>
          <Link href="/packages/import">
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
                <div>
                  <label htmlFor="package-name" className="text-sm font-medium">
                    Package Name
                  </label>
                  <Input
                    id="package-name"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder="e.g., requests"
                  />
                  {createErrors.name && (
                    <p className="text-xs text-destructive mt-1" role="alert">{createErrors.name}</p>
                  )}
                </div>
                <div>
                  <label htmlFor="package-registry" className="text-sm font-medium">
                    Registry
                  </label>
                  <Select
                    value={newRegistry}
                    onValueChange={(v) => { if (v) setNewRegistry(v as Registry) }}
                  >
                    <SelectTrigger id="package-registry">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="pypi">PyPI</SelectItem>
                      <SelectItem value="npm">npm</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <Button onClick={handleCreate} className="w-full" disabled={createMutation.isPending}>
                  Add Package
                </Button>
              </div>
            </DialogContent>
          </Dialog>
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
                placeholder="Search packages..."
                aria-label="Search packages"
              />
              <div className="space-y-1">
                <label htmlFor="packages-registry-filter" className="text-xs font-medium text-muted-foreground">
                  Registry
                </label>
                <Select
                  value={registryFilter || "all"}
                  onValueChange={(v) => setRegistryFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="packages-registry-filter" className="w-32">
                    <SelectValue placeholder="All" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="pypi">PyPI</SelectItem>
                    <SelectItem value="npm">npm</SelectItem>
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
                    onClick={() => router.push(`/packages/${row.original.id}`)}
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
                  icon={<Plus className="h-8 w-8" />}
                  title="No packages found."
                  description="Add your first package or sync the top packages to start monitoring."
                >
                  <Button size="sm" onClick={() => setDialogOpen(true)}>
                    Add Package
                  </Button>
                  <Button variant="outline" size="sm" onClick={handleSync} disabled={syncMutation.isPending}>
                    Sync Top Packages
                  </Button>
                </TableEmptyState>
              )}
            </TableBody>
          </Table>
          )}
          </div>

          <DataTablePagination table={table} total={total} />
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Package</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deleteTarget?.name}</strong>? This action
              cannot be undone. All releases and analysis data for this package will be permanently removed.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDelete}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
