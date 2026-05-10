"use client"

import { useState, useCallback, useEffect, useImperativeHandle, useRef, useMemo, type Ref } from "react"
import { useSearchParams } from "next/navigation"
import { useFilterParams } from "@/core"
import {
  useReactTable,
  getCoreRowModel,
  type PaginationState,
} from "@tanstack/react-table"
import { Card, CardContent, CardHeader, Button, Badge, Tabs, TabsList, TabsTrigger, TabsContent, DataTablePagination, TableSkeleton, TableError, EmptyState, Label } from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
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
  RotateCcw,
  Loader2,
  Clock,
  Activity,
  CheckCircle2,
  RefreshCw,
} from "lucide-react"
import { toast } from "sonner"
import type { QueueJob, QueueJobStatus, QueueJobType, QueueStats } from "@/domains/queue"
import { useQueueJobs, useRetryDeadJobs, useRetryDeadJob } from "@/features/settings/hooks/use-queue"
import { QueueJobDetailDialog } from "@/features/settings/ui/queue-job-detail-dialog"

export interface QueueJobsBrowserHandle {
  setFilter: (type: QueueJobType, status: QueueJobStatus) => void
}

interface QueueJobsBrowserProps {
  ref?: Ref<QueueJobsBrowserHandle>
  diffStats?: QueueStats | null
  analyzeStats?: QueueStats | null
}

const VALID_QUEUE_TYPES: QueueJobType[] = ["diff", "analyze"]
const VALID_STATUSES: QueueJobStatus[] = ["pending", "processing", "dead"]

const STUCK_THRESHOLD_SECONDS = 600 // 10 minutes

/** Returns the current time in seconds, updated every 10s. Avoids impure Date.now() in render. */
const useNowSeconds = (): number => {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000))
  useEffect(() => {
    const id = setInterval(() => setNow(Math.floor(Date.now() / 1000)), 10_000)
    return () => clearInterval(id)
  }, [])
  return now
}

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m}m ${s}s`
  }
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

const formatTimestamp = (unix: number): string => {
  if (!unix) return "-"
  return new Date(unix * 1000).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  })
}

export const QueueJobsBrowser = ({ diffStats, analyzeStats, ref }: QueueJobsBrowserProps) => {
    const containerRef = useRef<HTMLDivElement>(null)
    const searchParams = useSearchParams()

    const initialType = searchParams.get("type") ?? ""
    const initialStatus = searchParams.get("status") ?? ""

    const [queueType, setQueueType] = useState<QueueJobType>(
      VALID_QUEUE_TYPES.includes(initialType as QueueJobType) ? (initialType as QueueJobType) : "diff"
    )
    const [activeStatus, setActiveStatus] = useState<QueueJobStatus>(
      VALID_STATUSES.includes(initialStatus as QueueJobStatus) ? (initialStatus as QueueJobStatus) : "pending"
    )
    const [pagination, setPagination] = useState<PaginationState>({
      pageIndex: 0,
      pageSize: 20,
    })
    const [selectedJob, setSelectedJob] = useState<QueueJob | null>(null)
    const nowSeconds = useNowSeconds()

    // Sync filter state → URL search params
    // Defaults: type=diff, status=pending - omit from URL when they match
    const QUEUE_FILTER_DEFAULTS = useMemo(() => ({ type: "diff", status: "pending" }), [])
    useFilterParams(
      useMemo(() => ({
        type: queueType,
        status: activeStatus,
      }), [queueType, activeStatus]),
      QUEUE_FILTER_DEFAULTS,
    )

    const currentStats = queueType === "diff" ? diffStats : analyzeStats

    // Expose imperative handle for stats card clicks
    useImperativeHandle(ref, () => ({
      setFilter: (type: QueueJobType, status: QueueJobStatus) => {
        setQueueType(type)
        setActiveStatus(status)
        setPagination((prev) => ({ ...prev, pageIndex: 0 }))
        containerRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })
      },
    }))

    const {
      data: jobsRes,
      isLoading,
      isError,
      refetch,
    } = useQueueJobs({
      type: queueType,
      status: activeStatus,
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    })

    const retryAllMutation = useRetryDeadJobs()
    const retrySingleMutation = useRetryDeadJob()

    const jobs = jobsRes?.data ?? []
    const total = jobsRes?.meta?.total ?? 0

    const handleTabChange = useCallback((value: string | number | null) => {
      if (value !== null && typeof value === "string") {
        setActiveStatus(value as QueueJobStatus)
        setPagination((prev) => ({ ...prev, pageIndex: 0 }))
      }
    }, [])

    const handleTypeChange = useCallback((value: string | null) => {
      if (value !== null) {
        setQueueType(value as QueueJobType)
        setPagination((prev) => ({ ...prev, pageIndex: 0 }))
      }
    }, [])

    const handleRetryAll = async () => {
      try {
        const { data } = await retryAllMutation.mutateAsync(queueType)
        toast.success(`Retried ${data.count} dead job(s)`)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Failed to retry")
      }
    }

    const handleRetrySingle = async (jobId: string) => {
      try {
        await retrySingleMutation.mutateAsync(jobId)
        toast.success(`Job #${jobId} queued for retry`)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Failed to retry job")
      }
    }

    // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
    const table = useReactTable({
      data: jobs,
      columns: [],
      getCoreRowModel: getCoreRowModel(),
      manualPagination: true,
      pageCount: Math.ceil(total / pagination.pageSize),
      state: { pagination },
      onPaginationChange: setPagination,
    })

    const pendingCount = currentStats?.pending ?? 0
    const processingCount = currentStats?.processing ?? 0
    const deadCount = currentStats?.dead ?? 0

    return (
      <div ref={containerRef}>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Label htmlFor="queue-type-select">
                Queue:
              </Label>
              <Select value={queueType} onValueChange={handleTypeChange}>
                <SelectTrigger id="queue-type-select" className="w-32">
                  <SelectValue>
                    {queueType === "diff" ? "Diff" : "Analyze"}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="diff">Diff</SelectItem>
                  <SelectItem value="analyze">Analyze</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {activeStatus === "dead" && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleRetryAll}
                disabled={retryAllMutation.isPending || deadCount === 0}
              >
                {retryAllMutation.isPending ? (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                ) : (
                  <RotateCcw className="mr-2 size-4" />
                )}
                Retry All Dead
              </Button>
            )}
          </CardHeader>
          <CardContent className="space-y-4">
            <Tabs value={activeStatus} onValueChange={handleTabChange}>
              <TabsList variant="line">
                <TabsTrigger value="pending">
                  Pending ({pendingCount})
                </TabsTrigger>
                <TabsTrigger value="processing">
                  Processing ({processingCount})
                </TabsTrigger>
                <TabsTrigger value="dead">
                  Dead ({deadCount})
                </TabsTrigger>
              </TabsList>

              <TabsContent value={activeStatus}>
                {isLoading ? (
                  <TableSkeleton
                    columns={[
                      { width: "w-12", header: "ID" },
                      { width: "w-16", header: "Ref" },
                      { width: "w-16", header: "Attempts" },
                      { width: "w-32", header: activeStatus === "processing" ? "Duration" : "Last Error" },
                      { width: "w-24", header: activeStatus === "processing" ? "Started" : "Created" },
                    ]}
                    rows={5}
                  />
                ) : isError ? (
                  <TableError
                    colSpan={activeStatus === "dead" ? 6 : 5}
                    message="Failed to load queue jobs. Please try again."
                    onRetry={() => refetch()}
                  />
                ) : jobs.length === 0 ? (
                  <JobsEmptyState status={activeStatus} type={queueType} />
                ) : (
                  <>
                    {activeStatus === "pending" && (
                      <PendingTable jobs={jobs} onRowClick={setSelectedJob} />
                    )}
                    {activeStatus === "processing" && (
                      <ProcessingTable jobs={jobs} onRowClick={setSelectedJob} nowSeconds={nowSeconds} />
                    )}
                    {activeStatus === "dead" && (
                      <DeadTable
                        jobs={jobs}
                        onRowClick={setSelectedJob}
                        onRetry={handleRetrySingle}
                        isRetrying={retrySingleMutation.isPending}
                      />
                    )}
                    <DataTablePagination table={table} total={total} />
                  </>
                )}
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        <QueueJobDetailDialog
          job={selectedJob}
          onOpenChange={(open) => { if (!open) setSelectedJob(null) }}
          onRetry={handleRetrySingle}
          isRetrying={retrySingleMutation.isPending}
        />
      </div>
    )
  }

// ─── Empty States ─────────────────────────────────────────────

const JobsEmptyState = ({
  status,
  type,
}: {
  status: QueueJobStatus
  type: QueueJobType
}) => {
  if (status === "pending") {
    return (
      <EmptyState
        icon={<Clock className="h-8 w-8" />}
        title="No pending jobs."
        description={`The ${type} queue is empty. Jobs will appear here when new releases are detected.`}
      />
    )
  }
  if (status === "processing") {
    return (
      <EmptyState
        icon={<Activity className="h-8 w-8" />}
        title="No jobs processing."
        description={`No ${type} jobs are currently being processed.`}
      />
    )
  }
  return (
    <EmptyState
      icon={<CheckCircle2 className="h-8 w-8 text-green-600" />}
      title="No dead jobs."
      description={`All systems operational. No ${type} jobs have exceeded their max retry attempts.`}
    />
  )
}

// ─── Pending Table ─────────────────────────────────────────────

const PendingTable = ({
  jobs,
  onRowClick,
}: {
  jobs: QueueJob[]
  onRowClick: (job: QueueJob) => void
}) => (
  <div className="overflow-x-auto px-4">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>Package</TableHead>
          <TableHead>Attempts</TableHead>
          <TableHead className="hidden lg:table-cell">Last Error</TableHead>
          <TableHead className="hidden lg:table-cell">Created</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {jobs.map((job) => {
          const isRetrying = !!job.lastError
          return (
            <TableRow
              key={job.id}
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => onRowClick(job)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault()
                  onRowClick(job)
                }
              }}
            >
              <TableCell className="font-mono text-xs">{job.id}</TableCell>
              <TableCell className="text-xs">
                {job.metadata?.package ? (
                  <span>{job.metadata.package}{job.metadata.version && <span className="text-muted-foreground"> @{job.metadata.version}</span>}</span>
                ) : (
                  <span className="font-mono text-muted-foreground">#{job.referenceId}</span>
                )}
              </TableCell>
              <TableCell>
                <div className="flex flex-col gap-1">
                  <span className="text-sm">
                    {job.attempts}/{job.maxAttempts}
                  </span>
                  {isRetrying && (
                    <Badge
                      variant="outline"
                      className="w-fit border-orange-500/30 bg-orange-500/10 text-orange-600"
                    >
                      Retrying
                    </Badge>
                  )}
                </div>
              </TableCell>
              <TableCell className="hidden max-w-xs lg:table-cell">
                {job.lastError ? (
                  <span className="block truncate text-xs text-destructive">
                    {job.lastError}
                  </span>
                ) : (
                  <span className="text-xs text-muted-foreground">-</span>
                )}
              </TableCell>
              <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
                {formatTimestamp(job.createdAt)}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  </div>
)

// ─── Processing Table ─────────────────────────────────────────

const ProcessingTable = ({
  jobs,
  onRowClick,
  nowSeconds,
}: {
  jobs: QueueJob[]
  onRowClick: (job: QueueJob) => void
  nowSeconds: number
}) => (
  <div className="overflow-x-auto px-4">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>Package</TableHead>
          <TableHead className="hidden lg:table-cell">Attempts</TableHead>
          <TableHead>Duration</TableHead>
          <TableHead className="hidden lg:table-cell">Started</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {jobs.map((job) => {
          const duration = Math.max(0, nowSeconds - job.updatedAt)
          const isPossiblyStuck = duration > STUCK_THRESHOLD_SECONDS
          return (
            <TableRow
              key={job.id}
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => onRowClick(job)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault()
                  onRowClick(job)
                }
              }}
            >
              <TableCell className="font-mono text-xs">{job.id}</TableCell>
              <TableCell className="text-xs">
                {job.metadata?.package ? (
                  <span>{job.metadata.package}{job.metadata.version && <span className="text-muted-foreground"> @{job.metadata.version}</span>}</span>
                ) : (
                  <span className="font-mono text-muted-foreground">#{job.referenceId}</span>
                )}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                <span className="text-sm">
                  {job.attempts}/{job.maxAttempts}
                </span>
              </TableCell>
              <TableCell>
                <div className="flex flex-col gap-1">
                  <span className={`text-sm ${isPossiblyStuck ? "text-orange-600 font-medium" : ""}`}>
                    {formatDuration(duration)}
                  </span>
                  {isPossiblyStuck && (
                    <Badge
                      variant="outline"
                      className="w-fit animate-pulse border-orange-500/30 bg-orange-500/10 text-orange-600"
                    >
                      Possibly Stuck
                    </Badge>
                  )}
                </div>
              </TableCell>
              <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
                {formatTimestamp(job.updatedAt)}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  </div>
)

// ─── Dead Table ───────────────────────────────────────────────

const DeadTable = ({
  jobs,
  onRowClick,
  onRetry,
  isRetrying,
}: {
  jobs: QueueJob[]
  onRowClick: (job: QueueJob) => void
  onRetry: (jobId: string) => void
  isRetrying?: boolean
}) => (
  <div className="overflow-x-auto px-4">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>Package</TableHead>
          <TableHead className="hidden lg:table-cell">Attempts</TableHead>
          <TableHead>Last Error</TableHead>
          <TableHead className="hidden lg:table-cell">Died At</TableHead>
          <TableHead className="w-[1%] whitespace-nowrap text-right">
            <span className="sr-only">Actions</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {jobs.map((job) => (
          <TableRow
            key={job.id}
            className="cursor-pointer hover:bg-muted/50"
            onClick={() => onRowClick(job)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault()
                onRowClick(job)
              }
            }}
          >
            <TableCell className="font-mono text-xs">{job.id}</TableCell>
            <TableCell className="font-mono text-xs">#{job.referenceId}</TableCell>
            <TableCell className="hidden lg:table-cell">
              <span className="text-sm">
                {job.attempts}/{job.maxAttempts}
              </span>
            </TableCell>
            <TableCell className="max-w-xs">
              {job.lastError ? (
                <span className="block truncate text-xs text-destructive">
                  {job.lastError}
                </span>
              ) : (
                <span className="text-xs text-muted-foreground">-</span>
              )}
            </TableCell>
            <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
              {formatTimestamp(job.updatedAt)}
            </TableCell>
            <TableCell className="text-right">
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={(e) => {
                  e.stopPropagation()
                  onRetry(job.id)
                }}
                disabled={isRetrying}
                aria-label={`Retry job ${job.id}`}
              >
                <RefreshCw className="size-4" />
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  </div>
)
