"use client"

import { useState, useEffect, useMemo, useCallback } from "react"
import { useSortParams } from "@/core/hooks/use-sort-params"
import Link from "next/link"
import { Card, CardContent, CardHeader } from "@/ui/components/card"
import { Button } from "@/ui/components/button"
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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/components/alert-dialog"
import { TableSkeleton, type SkeletonColumn } from "@/ui/feedback/table-skeleton"
import { TableError } from "@/ui/feedback/table-error"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { DataTablePagination } from "@/ui/data/data-table-pagination"
import { SearchInput } from "@/ui/form/search-input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import { formatEcosystem } from "@/domains/common"
import { formatPopularity } from "@/domains/packages"
import type { Package } from "@/domains/packages"
import { SortableHeader } from "@/ui/data/sortable-header"
import { ArrowLeft, Check, X, CheckCheck, Radar, HelpCircle } from "lucide-react"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/ui/components/tooltip"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import "@/ui/data/table.types"
import {
  usePackageSuggestions,
  useApprovePackage,
  useRejectPackage,
  useBulkApprovePackages,
} from "@/features/packages/hooks/use-packages"

export const PackageSuggestionsView = () => {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [bulkAction, setBulkAction] = useState<"all" | "python" | "npm" | null>(null)
  const [search, setSearch] = useState("")
  const [debouncedSearch, setDebouncedSearch] = useState("")
  const [ecosystemFilter, setEcosystemFilter] = useState("")
  const [sorting, setSorting] = useSortParams()

  // Debounce search input to avoid excessive API calls
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Reset to first page when filters or sort change
  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [ecosystemFilter, sorting])

  // Server-side search/sort/filter/pagination
  const sort = sorting[0]
  const {
    data: suggestionsRes,
    isLoading,
    isError,
    refetch,
  } = usePackageSuggestions({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    search: debouncedSearch || undefined,
    ecosystem: ecosystemFilter || undefined,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const suggestions = suggestionsRes?.data ?? []
  const total = suggestionsRes?.meta?.total ?? 0

  const approveMutation = useApprovePackage()
  const rejectMutation = useRejectPackage()
  const bulkApproveMutation = useBulkApprovePackages()

  const handleApprove = useCallback(
    (id: number) => {
      approveMutation.mutate(id)
    },
    [approveMutation]
  )

  const handleReject = useCallback(
    (id: number) => {
      rejectMutation.mutate(id)
    },
    [rejectMutation]
  )

  const confirmBulkApprove = useCallback(() => {
    if (!bulkAction) return
    if (bulkAction === "all") {
      const ids = suggestions.map((s) => s.id)
      if (ids.length > 0) bulkApproveMutation.mutate({ packageIds: ids })
    } else {
      bulkApproveMutation.mutate({ ecosystem: bulkAction })
    }
    setBulkAction(null)
  }, [bulkAction, suggestions, bulkApproveMutation])

  const bulkActionLabel = useMemo(() => {
    if (bulkAction === "all") return "all pending suggestions"
    if (bulkAction === "python") return "all Python suggestions"
    if (bulkAction === "npm") return "all NPM suggestions"
    return ""
  }, [bulkAction])

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Name" },
    { width: "w-16", header: "Ecosystem" },
    { width: "w-12", header: "Rank" },
    { width: "w-20", header: "Popularity" },
    { width: "w-16", header: "" },
  ]

  const columns = useMemo<ColumnDef<Package>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <SortableHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <span className="font-medium">{row.original.name}</span>
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
        accessorKey: "rank",
        header: "Rank",
        enableSorting: false,
        cell: ({ row }) => row.original.rank ?? "\u2014",
      },
      {
        id: "popularity",
        accessorKey: "downloadCount",
        header: ({ column }) => (
          <div className="flex items-center gap-1">
            <SortableHeader column={column} title="Popularity" />
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger render={<span><HelpCircle className="h-3.5 w-3.5 text-muted-foreground" /></span>} />
                <TooltipContent>
                  <p className="font-medium mb-1">Different metrics per ecosystem:</p>
                  <ul className="text-xs space-y-0.5">
                    <li><strong>Python (PyPI)</strong> — Downloads per month</li>
                    <li><strong>NPM</strong> — Popularity score (0–1)</li>
                  </ul>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        ),
        cell: ({ row }) => {
          const pkg = row.original
          return (
            <span className="text-sm tabular-nums">
              {formatPopularity(pkg.ecosystem, pkg.downloadCount, pkg.popularityScore)}
            </span>
          )
        },
      },
      {
        id: "actions",
        header: () => <span className="sr-only">Actions</span>,
        enableSorting: false,
        meta: { headerClassName: "w-[1%] whitespace-nowrap text-right", cellClassName: "text-right" },
        cell: ({ row }) => {
          const pkg = row.original
          return (
            <div className="flex items-center justify-end gap-1">
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Approve ${pkg.name}`}
                onClick={() => handleApprove(pkg.id)}
                disabled={approveMutation.isPending}
              >
                <Check className="size-3.5 text-emerald-600 dark:text-emerald-500 mr-1" />
                Approve
              </Button>
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Reject ${pkg.name}`}
                onClick={() => handleReject(pkg.id)}
                disabled={rejectMutation.isPending}
              >
                <X className="size-3.5 text-destructive mr-1" />
                Reject
              </Button>
            </div>
          )
        },
      },
    ],
    [handleApprove, handleReject, approveMutation.isPending, rejectMutation.isPending]
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  const table = useReactTable({
    data: suggestions,
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
      <div>
        <Link
          href="/packages"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Packages
        </Link>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-3xl font-bold">Suggestions</h1>
            {total > 0 && (
              <Badge variant="secondary">{total} pending</Badge>
            )}
          </div>
          {total > 0 && (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setBulkAction("python")}
                disabled={bulkApproveMutation.isPending}
              >
                <CheckCheck className="h-4 w-4 mr-1" />
                Approve All Python
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setBulkAction("npm")}
                disabled={bulkApproveMutation.isPending}
              >
                <CheckCheck className="h-4 w-4 mr-1" />
                Approve All NPM
              </Button>
              <Button
                size="sm"
                onClick={() => setBulkAction("all")}
                disabled={bulkApproveMutation.isPending}
              >
                <CheckCheck className="h-4 w-4 mr-1" />
                Approve All
              </Button>
            </div>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-end gap-4">
            <SearchInput
              value={search}
              onChange={setSearch}
              onClear={() => setSearch("")}
              placeholder="Search suggestions..."
              aria-label="Search suggestions"
            />
            <div className="space-y-1">
              <label htmlFor="suggestions-ecosystem-filter" className="text-xs font-medium text-muted-foreground">
                Ecosystem
              </label>
              <Select
                value={ecosystemFilter || "all"}
                onValueChange={(v) => {
                  setEcosystemFilter(v === "all" ? "" : (v ?? ""))
                  setPagination((prev) => ({ ...prev, pageIndex: 0 }))
                }}
              >
                <SelectTrigger id="suggestions-ecosystem-filter" className="w-32">
                  <SelectValue>{ecosystemFilter === "python" ? "Python" : ecosystemFilter === "npm" ? "NPM" : "All"}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  <SelectItem value="python">Python</SelectItem>
                  <SelectItem value="npm">NPM</SelectItem>
                </SelectContent>
              </Select>
            </div>
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
                      <TableRow key={row.id}>
                        {row.getVisibleCells().map((cell) => (
                          <TableCell key={cell.id} className={cell.column.columnDef.meta?.cellClassName}>
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : (
                    <TableEmptyState
                      colSpan={columns.length}
                      icon={<Radar className="h-8 w-8" />}
                      title="No pending suggestions."
                      description="Run discovery to find new packages to monitor."
                    />
                  )}
                </TableBody>
              </Table>
            )}
          </div>
          <DataTablePagination table={table} total={total} />
        </CardContent>
      </Card>

      {/* Bulk Approve Confirmation */}
      <AlertDialog open={!!bulkAction} onOpenChange={(open) => { if (!open) setBulkAction(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Bulk Approve</AlertDialogTitle>
            <AlertDialogDescription>
              Approve {bulkActionLabel}? This will add them to active monitoring.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmBulkApprove}>
              Approve
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
