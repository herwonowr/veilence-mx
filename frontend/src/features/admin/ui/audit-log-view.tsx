"use client"

import { useState, useMemo, useEffect } from "react"
import Link from "next/link"
import { useParams, useRouter, useSearchParams } from "next/navigation"
import { useFilterParams, useDebouncedValue, useSortParams, ROUTES, cn } from "@/core"
import {
  Button,
  buttonVariants,
  Input,
  Label,
  Badge,
  TableSkeleton,
  TableEmptyState,
  FilterChips,
  DataTablePagination,
  DataTableColumnToggle,
  SortableHeader,
  Calendar,
  Popover,
  PopoverContent,
  PopoverTrigger,
  AuditLogDetailDialog,
  type ActiveFilter,
  type SkeletonColumn,
} from "@/ui"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import { ArrowLeft, CalendarIcon, Filter, ScrollText } from "lucide-react"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
  type VisibilityState,
} from "@tanstack/react-table"
import "@/ui/data/table.types"
import { useAuditLogs } from "@/features/admin/hooks/use-workspaces"
import type { AuditLog } from "@/domains/admin"

const formatStartOfDay = (d: Date): string => {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, "0")
  const dd = String(d.getDate()).padStart(2, "0")
  return `${yyyy}-${mm}-${dd}T00:00:00Z`
}

const formatEndOfDay = (d: Date): string => {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, "0")
  const dd = String(d.getDate()).padStart(2, "0")
  return `${yyyy}-${mm}-${dd}T23:59:59Z`
}

const parseValidDate = (value: string | null): Date | undefined => {
  if (!value) return undefined
  const parsed = new Date(value)
  if (isNaN(parsed.getTime())) return undefined
  return parsed
}

export const AuditLogView = () => {
  const params = useParams<{ id: string }>()
  const workspaceId = params.id
  const validWorkspaceId = workspaceId ?? ""
  const router = useRouter()
  const searchParams = useSearchParams()

  const initialAction = searchParams.get("action") ?? ""
  const initialResource = searchParams.get("resource") ?? ""
  const initialFrom = parseValidDate(searchParams.get("from"))
  const initialTo = parseValidDate(searchParams.get("to"))

  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null)
  const [action, setAction] = useState(initialAction)
  const [resource, setResource] = useState(initialResource)
  const debouncedAction = useDebouncedValue(action, 300)
  const debouncedResource = useDebouncedValue(resource, 300)
  const [fromDate, setFromDate] = useState<Date | undefined>(initialFrom)
  const [toDate, setToDate] = useState<Date | undefined>(initialTo)
  const [fromOpen, setFromOpen] = useState(false)
  const [toOpen, setToOpen] = useState(false)

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({
    details: false,
  })

  useFilterParams(
    useMemo(() => ({
      action: debouncedAction,
      resource: debouncedResource,
      from: fromDate ? formatStartOfDay(fromDate) : "",
      to: toDate ? formatEndOfDay(toDate) : "",
    }), [debouncedAction, debouncedResource, fromDate, toDate]),
  )

  const handleFromSelect = (date: Date | undefined) => {
    setFromDate(date)
    setFromOpen(false)
  }

  const handleToSelect = (date: Date | undefined) => {
    setToDate(date)
    setToOpen(false)
  }

  const hasActiveFilters = !!(action || resource || fromDate || toDate)

  const clearAllFilters = () => {
    setAction("")
    setResource("")
    setFromDate(undefined)
    setToDate(undefined)
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(action
      ? [{ label: "Action", value: action, onRemove: () => setAction("") }]
      : []),
    ...(resource
      ? [{ label: "Resource", value: resource, onRemove: () => setResource("") }]
      : []),
    ...(fromDate
      ? [{ label: "From", value: fromDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }), onRemove: () => { setFromDate(undefined); setToDate(undefined) } }]
      : []),
    ...(toDate
      ? [{ label: "To", value: toDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }), onRemove: () => setToDate(undefined) }]
      : []),
  ]

  const sort = sorting[0]
  const { data: logsRes, isLoading, isError, refetch } = useAuditLogs(validWorkspaceId, {
    action: debouncedAction || undefined,
    resource: debouncedResource || undefined,
    from_date: fromDate ? formatStartOfDay(fromDate) : undefined,
    to_date: toDate ? formatEndOfDay(toDate) : undefined,
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    sort_by: sort?.id,
    sort_dir: sort ? (sort.desc ? "desc" : "asc") : undefined,
  })

  const logs = logsRes?.data ?? []
  const total = logsRes?.meta?.total ?? 0

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [debouncedAction, debouncedResource, fromDate, toDate, sorting])

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Timestamp" },
    { width: "w-16", header: "Action" },
    { width: "w-20", header: "Resource" },
    { width: "w-40", header: "Details" },
    { width: "w-24", header: "IP Address" },
  ]

  const columns = useMemo<ColumnDef<AuditLog>[]>(
    () => [
      {
        accessorKey: "createdAt",
        header: ({ column }) => <SortableHeader column={column} title="Timestamp" />,
        enableHiding: false,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground whitespace-nowrap">
            {new Date(row.original.createdAt).toLocaleString()}
          </span>
        ),
      },
      {
        accessorKey: "action",
        header: ({ column }) => <SortableHeader column={column} title="Action" />,
        enableHiding: false,
        cell: ({ row }) => (
          <Badge variant="outline">{row.original.action}</Badge>
        ),
      },
      {
        accessorKey: "resource",
        header: ({ column }) => <SortableHeader column={column} title="Resource" />,
        enableHiding: false,
        cell: ({ row }) => (
          <Badge variant="secondary" className="w-fit capitalize">
            {row.original.resource.replace(/_/g, " ")}
          </Badge>
        ),
      },
      {
        accessorKey: "details",
        header: "Details",
        enableSorting: false,
        cell: ({ row }) => (
          <span className="max-w-sm text-sm truncate block">
            {row.original.details || "-"}
          </span>
        ),
      },
      {
        accessorKey: "ipAddress",
        header: "IP Address",
        enableSorting: false,
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original.ipAddress || "-"}
          </span>
        ),
      },
    ],
    []
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
  const table = useReactTable({
    data: logs,
    columns,
    pageCount,
    state: { pagination, sorting, columnVisibility },
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
  })

  if (!workspaceId) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-lg font-medium text-destructive">Invalid workspace ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push(ROUTES.WORKSPACES)}>
          Back to Workspaces
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href={`/workspaces/${workspaceId}`}
          className="w-fit text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Workspace
        </Link>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <ScrollText className="size-7" />
          Audit Log
        </h1>
        <p className="text-sm text-muted-foreground">
          View activity history for this workspace
        </p>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base flex items-center gap-2">
            <Filter className="size-4" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="space-y-1">
              <Label htmlFor="filter-action" className="text-xs">
                Action
              </Label>
              <Input
                id="filter-action"
                placeholder="e.g. create, update"
                value={action}
                onChange={(e) => setAction(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="filter-resource" className="text-xs">
                Resource
              </Label>
              <Input
                id="filter-resource"
                placeholder="e.g. package, member"
                value={resource}
                onChange={(e) => setResource(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">From</Label>
              <Popover open={fromOpen} onOpenChange={setFromOpen}>
                <PopoverTrigger
                  render={
                    <button
                      type="button"
                      className={cn(
                        buttonVariants({ variant: "outline" }),
                        "w-full justify-start text-left font-normal h-8"
                      )}
                      aria-label="Select start date"
                    />
                  }
                >
                  <CalendarIcon className="mr-2 h-4 w-4 text-muted-foreground" />
                  {fromDate ? (
                    fromDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
                  ) : (
                    <span className="text-muted-foreground">Pick a date</span>
                  )}
                </PopoverTrigger>
                <PopoverContent align="start" className="w-auto">
                  <Calendar
                    mode="single"
                    selected={fromDate}
                    onSelect={handleFromSelect}
                    disabled={(date) => toDate ? date > toDate : date > new Date()}
                    defaultMonth={fromDate}
                  />
                </PopoverContent>
              </Popover>
            </div>
            <div className="space-y-1">
              <Label className="text-xs">To</Label>
              <Popover open={toOpen} onOpenChange={fromDate ? setToOpen : undefined}>
                <PopoverTrigger
                  render={
                    <button
                      type="button"
                      disabled={!fromDate}
                      className={cn(
                        buttonVariants({ variant: "outline" }),
                        "w-full justify-start text-left font-normal h-8",
                        !fromDate && "opacity-50 cursor-not-allowed"
                      )}
                      aria-label="Select end date"
                    />
                  }
                >
                  <CalendarIcon className="mr-2 h-4 w-4 text-muted-foreground" />
                  {toDate ? (
                    toDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
                  ) : (
                    <span className="text-muted-foreground">Pick a date</span>
                  )}
                </PopoverTrigger>
                <PopoverContent align="start" className="w-auto">
                  <Calendar
                    mode="single"
                    selected={toDate}
                    onSelect={handleToSelect}
                    disabled={(date) => fromDate ? date < fromDate || date > new Date() : date > new Date()}
                    defaultMonth={toDate ?? fromDate}
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>
          {hasActiveFilters && (
            <div className="mt-3">
              <FilterChips filters={activeFilters} onClearAll={clearAllFilters} />
            </div>
          )}
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardContent>
          <div className="flex justify-end mb-3">
            <DataTableColumnToggle table={table} />
          </div>
          <div className="overflow-x-auto">
            {isLoading ? (
              <TableSkeleton columns={skeletonColumns} rows={5} />
            ) : isError ? (
              <TableEmptyState
                colSpan={columns.length}
                icon={<ScrollText className="h-8 w-8" />}
                title="Failed to load audit logs."
                description="An error occurred while fetching data."
              >
                <Button variant="outline" size="sm" onClick={() => refetch()}>
                  Retry
                </Button>
              </TableEmptyState>
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
                      <TableRow
                        key={row.id}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => setSelectedLog(row.original)}
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
                        icon={<ScrollText className="h-8 w-8" />}
                        title="No matching audit logs."
                        description="Try adjusting your filters."
                      >
                        <Button variant="outline" size="sm" onClick={clearAllFilters}>
                          Clear filters
                        </Button>
                      </TableEmptyState>
                    ) : (
                      <TableEmptyState
                        colSpan={columns.length}
                        icon={<ScrollText className="h-8 w-8" />}
                        title="No audit logs found."
                        description="Activity history will appear here as actions are performed in this workspace."
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

      <AuditLogDetailDialog
        log={selectedLog}
        onOpenChange={(open) => { if (!open) setSelectedLog(null) }}
      />
    </div>
  )
}
