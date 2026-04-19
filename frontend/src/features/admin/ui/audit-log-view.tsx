"use client"

import { useState, useMemo } from "react"
import Link from "next/link"
import { useParams, useRouter, useSearchParams } from "next/navigation"
import { useFilterParams, useDebouncedValue } from "@/core"
import { Button, buttonVariants, Input, Label, Badge, Skeleton, TableEmptyState, FilterChips, Calendar, Popover, PopoverContent, PopoverTrigger, type ActiveFilter } from "@/ui"
import { cn } from "@/core"
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
import {
  ArrowLeft,
  CalendarIcon,
  ChevronLeft,
  ChevronRight,
  Filter,
  ScrollText,
} from "lucide-react"
import { useAuditLogs } from "@/features/admin/hooks/use-workspaces"

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
  const workspaceId = parseInt(params.id, 10)
  const validWorkspaceId = isNaN(workspaceId) ? 0 : workspaceId
  const router = useRouter()
  const searchParams = useSearchParams()

  // Read URL search params as initial filter values
  const initialAction = searchParams.get("action") ?? ""
  const initialResource = searchParams.get("resource") ?? ""
  const initialFrom = parseValidDate(searchParams.get("from"))
  const initialTo = parseValidDate(searchParams.get("to"))

  // Filters
  const [action, setAction] = useState(initialAction)
  const [resource, setResource] = useState(initialResource)
  const debouncedAction = useDebouncedValue(action, 300)
  const debouncedResource = useDebouncedValue(resource, 300)
  const [fromDate, setFromDate] = useState<Date | undefined>(initialFrom)
  const [toDate, setToDate] = useState<Date | undefined>(initialTo)
  const [fromOpen, setFromOpen] = useState(false)
  const [toOpen, setToOpen] = useState(false)
  const [page, setPage] = useState(1)
  const limit = 20

  // Sync filter state → URL search params
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
    setPage(1)
  }

  const handleToSelect = (date: Date | undefined) => {
    setToDate(date)
    setToOpen(false)
    setPage(1)
  }

  const hasActiveFilters = !!(action || resource || fromDate || toDate)

  const clearAllFilters = () => {
    setAction("")
    setResource("")
    setFromDate(undefined)
    setToDate(undefined)
    setPage(1)
  }

  const activeFilters: ActiveFilter[] = [
    ...(action
      ? [{ label: "Action", value: action, onRemove: () => { setAction(""); setPage(1) } }]
      : []),
    ...(resource
      ? [{ label: "Resource", value: resource, onRemove: () => { setResource(""); setPage(1) } }]
      : []),
    ...(fromDate
      ? [{ label: "From", value: fromDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }), onRemove: () => { setFromDate(undefined); setToDate(undefined); setPage(1) } }]
      : []),
    ...(toDate
      ? [{ label: "To", value: toDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }), onRemove: () => { setToDate(undefined); setPage(1) } }]
      : []),
  ]

  const { data: logsRes, isLoading } = useAuditLogs(validWorkspaceId, {
    action: debouncedAction || undefined,
    resource: debouncedResource || undefined,
    from_date: fromDate ? formatStartOfDay(fromDate) : undefined,
    to_date: toDate ? formatEndOfDay(toDate) : undefined,
    page,
    limit,
  })

  const logs = logsRes?.data ?? []
  const meta = logsRes?.meta ?? null
  const totalPages = meta ? Math.ceil(meta.total / meta.limit) : 1

  if (isNaN(workspaceId)) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-lg font-medium text-destructive">Invalid workspace ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/workspaces")}>
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
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
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
                onChange={(e) => {
                  setAction(e.target.value)
                  setPage(1)
                }}
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
                onChange={(e) => {
                  setResource(e.target.value)
                  setPage(1)
                }}
              />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">
                From
              </Label>
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
              <Label className="text-xs">
                To
              </Label>
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
        <CardContent className="p-0">
          {isLoading ? (
            <div className="space-y-2 p-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Timestamp</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Resource</TableHead>
                  <TableHead>Details</TableHead>
                  <TableHead>IP Address</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {new Date(log.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{log.action}</Badge>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-sm">
                        {log.resource}
                        {log.resourceId ? `#${log.resourceId}` : ""}
                      </span>
                    </TableCell>
                    <TableCell className="max-w-xs truncate text-sm">
                      {log.details || "-"}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {log.ipAddress || "-"}
                    </TableCell>
                  </TableRow>
                ))}
                {logs.length === 0 && (
                  hasActiveFilters ? (
                    <TableEmptyState
                      colSpan={5}
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
                    colSpan={5}
                    icon={<ScrollText className="h-8 w-8" />}
                    title="No audit logs found."
                    description="Activity history will appear here as actions are performed in this workspace."
                  />
                  )
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {meta && totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {(page - 1) * limit + 1}–
            {Math.min(page * limit, meta.total)} of {meta.total} entries
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              <ChevronLeft className="size-4" />
              Previous
            </Button>
            <span className="text-sm text-muted-foreground">
              Page {page} of {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
              <ChevronRight className="size-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

