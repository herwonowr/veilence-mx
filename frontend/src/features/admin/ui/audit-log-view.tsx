"use client"

import { useState } from "react"
import Link from "next/link"
import { useParams, useRouter } from "next/navigation"
import { Button, buttonVariants } from "@/ui/components/button"
import { cn } from "@/core/utils"
import { Input } from "@/ui/components/input"
import { Label } from "@/ui/components/label"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/ui/components/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui/components/table"
import { Badge } from "@/ui/components/badge"
import { Skeleton } from "@/ui/components/skeleton"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { Calendar } from "@/ui/components/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/components/popover"
import {
  ArrowLeft,
  CalendarIcon,
  ChevronLeft,
  ChevronRight,
  Filter,
  ScrollText,
} from "lucide-react"
import { useAuditLogs } from "@/features/admin/hooks/use-organizations"
import { useDebouncedValue } from "@/core/hooks/use-debounced-value"

const formatDate = (d: Date): string => d.toISOString().split("T")[0]

export const AuditLogView = () => {
  const params = useParams<{ id: string }>()
  const orgId = parseInt(params.id, 10)
  const validOrgId = isNaN(orgId) ? 0 : orgId
  const router = useRouter()

  // Filters
  const [action, setAction] = useState("")
  const [resource, setResource] = useState("")
  const debouncedAction = useDebouncedValue(action, 300)
  const debouncedResource = useDebouncedValue(resource, 300)
  const [fromDate, setFromDate] = useState<Date | undefined>()
  const [toDate, setToDate] = useState<Date | undefined>()
  const [fromOpen, setFromOpen] = useState(false)
  const [toOpen, setToOpen] = useState(false)
  const [page, setPage] = useState(1)
  const limit = 20

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

  const { data: logsRes, isLoading } = useAuditLogs(validOrgId, {
    action: debouncedAction || undefined,
    resource: debouncedResource || undefined,
    from: fromDate ? formatDate(fromDate) : undefined,
    to: toDate ? formatDate(toDate) : undefined,
    page,
    limit,
  })

  const logs = logsRes?.data ?? []
  const meta = logsRes?.meta ?? null
  const totalPages = meta ? Math.ceil(meta.total / meta.limit) : 1

  if (isNaN(orgId)) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-lg font-medium text-destructive">Invalid organization ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/organizations")}>
          Back to Organizations
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href={`/organizations/${orgId}`}
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Organization
        </Link>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <ScrollText className="size-7" />
          Audit Log
        </h1>
        <p className="text-sm text-muted-foreground">
          View activity history for this organization
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
              <Popover open={toOpen} onOpenChange={setToOpen}>
                <PopoverTrigger
                  render={
                    <button
                      type="button"
                      className={cn(
                        buttonVariants({ variant: "outline" }),
                        "w-full justify-start text-left font-normal h-8"
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
                      {log.details || "—"}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {log.ipAddress || "—"}
                    </TableCell>
                  </TableRow>
                ))}
                {logs.length === 0 && (
                  <TableEmptyState
                    colSpan={5}
                    icon={<ScrollText className="h-8 w-8" />}
                    title="No audit logs found."
                    description="Activity history will appear here as actions are performed in this organization."
                  />
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

