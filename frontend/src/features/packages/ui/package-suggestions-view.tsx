"use client"

import { useState, useEffect, useMemo, useCallback } from "react"
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
import { formatPopularity, popularityLabel } from "@/domains/packages"
import type { Package } from "@/domains/packages"
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
  const [sortField, setSortField] = useState<"name" | "ecosystem" | "popularity" | "">("")
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc")

  // Debounce search input to avoid excessive API calls
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  // Server-side search/sort/filter/pagination
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
    sortBy: sortField || undefined,
    sortDir: sortField ? sortDir : undefined,
  })

  const suggestions = suggestionsRes?.data ?? []
  const total = suggestionsRes?.meta?.total ?? 0

  const approveMutation = useApprovePackage()
  const rejectMutation = useRejectPackage()
  const bulkApproveMutation = useBulkApprovePackages()

  const toggleSort = useCallback((field: "name" | "ecosystem" | "popularity") => {
    if (sortField === field) {
      setSortDir((d) => d === "asc" ? "desc" : "asc")
    } else {
      setSortField(field)
      setSortDir("asc")
    }
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [sortField])

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
    { width: "w-16", header: "Actions" },
  ]

  const sortIndicator = useCallback((field: string) => {
    if (sortField !== field) return ""
    return sortDir === "asc" ? " ↑" : " ↓"
  }, [sortField, sortDir])

  const columns = useMemo<ColumnDef<Package>[]>(
    () => [
      {
        accessorKey: "name",
        header: () => (
          <button type="button" className="flex items-center gap-1 hover:text-foreground" onClick={() => toggleSort("name")}>
            Name{sortIndicator("name")}
          </button>
        ),
        cell: ({ row }) => (
          <span className="font-medium">{row.original.name}</span>
        ),
      },
      {
        accessorKey: "ecosystem",
        header: () => (
          <button type="button" className="flex items-center gap-1 hover:text-foreground" onClick={() => toggleSort("ecosystem")}>
            Ecosystem{sortIndicator("ecosystem")}
          </button>
        ),
        cell: ({ row }) => (
          <Badge variant="outline">{formatEcosystem(row.original.ecosystem)}</Badge>
        ),
      },
      {
        accessorKey: "rank",
        header: "Rank",
        cell: ({ row }) => row.original.rank ?? "\u2014",
      },
      {
        id: "popularity",
        header: () => (
          <div className="flex items-center gap-1">
            <button type="button" className="hover:text-foreground" onClick={() => toggleSort("popularity")}>
              Popularity{sortIndicator("popularity")}
            </button>
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
        header: "Actions",
        size: 180,
        cell: ({ row }) => {
          const pkg = row.original
          return (
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Approve ${pkg.name}`}
                onClick={() => handleApprove(pkg.id)}
                disabled={approveMutation.isPending}
              >
                <Check className="h-4 w-4 mr-1" />
                Approve
              </Button>
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Reject ${pkg.name}`}
                onClick={() => handleReject(pkg.id)}
                disabled={rejectMutation.isPending}
              >
                <X className="h-4 w-4 mr-1" />
                Reject
              </Button>
            </div>
          )
        },
      },
    ],
    [handleApprove, handleReject, approveMutation.isPending, rejectMutation.isPending, toggleSort, sortIndicator]
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  const table = useReactTable({
    data: suggestions,
    columns,
    pageCount,
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
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
