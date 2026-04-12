"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/ui/components/card"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import {
  RefreshCw,
  RotateCcw,
  Loader2,
  Activity,
  Clock,
  CheckCircle2,
  XCircle,
  Skull,
  ListOrdered,
} from "lucide-react"
import { EmptyState } from "@/ui/feedback/empty-state"
import { useQueueStats, useDeadJobs, useRetryDeadJobs, useRetryDeadJob } from "@/features/settings/hooks/use-queue"
import type { QueueStats } from "@/domains/queue"
import { toast } from "sonner"

export const QueueView = () => {
  const [deadJobType, setDeadJobType] = useState<string | undefined>(undefined)

  const {
    data: queueRes,
    isLoading: statsLoading,
    refetch: refetchStats,
    isRefetching: statsRefetching,
  } = useQueueStats({ refetchInterval: 10_000 })

  const {
    data: deadRes,
    isLoading: deadLoading,
    refetch: refetchDead,
  } = useDeadJobs(deadJobType, { refetchInterval: 10_000 })

  const retryMutation = useRetryDeadJobs()
  const retrySingleMutation = useRetryDeadJob()

  const stats = queueRes?.data ?? null
  const deadJobs = deadRes?.data ?? []

  const handleRetry = async (type?: string) => {
    try {
      const { data } = await retryMutation.mutateAsync(type)
      toast.success(`Retried ${data.count} dead job(s)`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to retry")
    }
  }

  const handleRetrySingle = async (jobId: string) => {
    try {
      await retrySingleMutation.mutateAsync(jobId)
      toast.success("Job queued for retry")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to retry job")
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Queue Monitor</h1>
          <p className="mt-1 text-muted-foreground">
            Real-time view of diff and analysis job queues. Auto-refreshes every
            10 seconds.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            refetchStats()
            refetchDead()
          }}
          disabled={statsRefetching}
        >
          <RefreshCw
            className={`mr-2 size-4 ${statsRefetching ? "animate-spin" : ""}`}
          />
          Refresh
        </Button>
      </div>

      {/* Queue Stats Cards */}
      {statsLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : stats ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <QueueStatsCard title="Diff Queue" type="diff" stats={stats.diff} />
          <QueueStatsCard
            title="Analyze Queue"
            type="analyze"
            stats={stats.analyze}
          />
        </div>
      ) : (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Queue stats unavailable. Redis may not be running.
          </CardContent>
        </Card>
      )}

      {/* Dead Jobs Section */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Skull className="size-5" />
              Dead Jobs
            </CardTitle>
            <CardDescription>
              Jobs that exceeded max retry attempts. Inspect and retry as needed.
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Select
              value={deadJobType ?? "all"}
              onValueChange={(v) =>
                setDeadJobType(v === "all" ? undefined : (v ?? undefined))
              }
            >
              <SelectTrigger className="w-28">
                <SelectValue>{deadJobType ? deadJobType.charAt(0).toUpperCase() + deadJobType.slice(1) : "All"}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="diff">Diff</SelectItem>
                <SelectItem value="analyze">Analyze</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleRetry(deadJobType)}
              disabled={retryMutation.isPending || deadJobs.length === 0}
            >
              {retryMutation.isPending ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RotateCcw className="mr-2 size-4" />
              )}
              Retry All
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {deadLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          ) : deadJobs.length === 0 ? (
            <EmptyState
              icon={<CheckCircle2 className="h-8 w-8 text-green-600" />}
              title="No dead jobs."
              description="All systems operational. No jobs have exceeded their max retry attempts."
            />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Reference</TableHead>
                    <TableHead>Attempts</TableHead>
                    <TableHead>Last Error</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="w-16">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deadJobs.map((job) => (
                    <TableRow key={job.id}>
                      <TableCell className="font-mono text-xs">
                        {job.id.slice(0, 8)}...
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="capitalize">
                          {job.type}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        #{job.referenceId}
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {job.attempts}/{job.maxAttempts}
                        </span>
                      </TableCell>
                      <TableCell className="max-w-xs truncate text-xs text-destructive">
                        {job.lastError ?? "-"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {new Date(job.createdAt * 1000).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => handleRetrySingle(job.id)}
                          disabled={retrySingleMutation.isPending}
                          aria-label={`Retry job ${job.id.slice(0, 8)}`}
                        >
                          <RefreshCw className="size-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

// ─── Queue Stats Card ──────────────────────────────────────────

const QueueStatsCard = ({
  title,
  stats,
}: {
  title: string
  type: string
  stats: QueueStats
}) => {
  const total =
    stats.pending + stats.processing + stats.completed + stats.failed + stats.dead

  const items = [
    {
      label: "Pending",
      value: stats.pending,
      icon: <Clock className="size-4 text-muted-foreground" />,
      color: "text-muted-foreground",
    },
    {
      label: "Processing",
      value: stats.processing,
      icon: <Activity className="size-4 text-blue-500" />,
      color: "text-blue-500",
    },
    {
      label: "Completed",
      value: stats.completed,
      icon: <CheckCircle2 className="size-4 text-green-500" />,
      color: "text-green-500",
    },
    {
      label: "Failed",
      value: stats.failed,
      icon: <XCircle className="size-4 text-orange-500" />,
      color: "text-orange-500",
    },
    {
      label: "Dead",
      value: stats.dead,
      icon: <Skull className="size-4 text-destructive" />,
      color: "text-destructive",
    },
  ]

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <ListOrdered className="size-4" />
          {title}
        </CardTitle>
        <p className="text-xs text-muted-foreground">{total} total jobs</p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {items.map((item) => (
            <div
              key={item.label}
              className="flex items-center justify-between"
            >
              <div className="flex items-center gap-2 text-sm">
                {item.icon}
                <span>{item.label}</span>
              </div>
              <span className={`text-lg font-semibold tabular-nums ${item.color}`}>
                {item.value.toLocaleString()}
              </span>
            </div>
          ))}
        </div>

        {/* Progress bar */}
        {total > 0 && (
          <div className="mt-4 flex h-2 overflow-hidden rounded-full bg-muted">
            {stats.completed > 0 && (
              <div
                className="bg-green-500"
                style={{ width: `${(stats.completed / total) * 100}%` }}
              />
            )}
            {stats.processing > 0 && (
              <div
                className="bg-blue-500"
                style={{ width: `${(stats.processing / total) * 100}%` }}
              />
            )}
            {stats.pending > 0 && (
              <div
                className="bg-muted-foreground/30"
                style={{ width: `${(stats.pending / total) * 100}%` }}
              />
            )}
            {stats.failed > 0 && (
              <div
                className="bg-orange-500"
                style={{ width: `${(stats.failed / total) * 100}%` }}
              />
            )}
            {stats.dead > 0 && (
              <div
                className="bg-destructive"
                style={{ width: `${(stats.dead / total) * 100}%` }}
              />
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
