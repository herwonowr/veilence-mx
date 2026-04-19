"use client"

import { useRef, useState } from "react"
import { Card, CardContent, Button } from "@/ui"
import { RefreshCw, Loader2, Play, RotateCcw } from "lucide-react"
import { useQueueStats, useRetryDeadJobs, queueKeys } from "@/features/settings/hooks/use-queue"
import { useReanalyzeAll } from "@/features/settings/hooks/use-settings"
import { useQueryClient } from "@tanstack/react-query"
import { QueueStatsCard } from "@/features/settings/ui/queue-stats-card"
import { QueueJobsBrowser, type QueueJobsBrowserHandle } from "@/features/settings/ui/queue-jobs-browser"
import type { QueueJobStatus, QueueJobType } from "@/domains/queue"

export const QueueView = () => {
  const jobsBrowserRef = useRef<QueueJobsBrowserHandle>(null)
  const queryClient = useQueryClient()
  const [queueMessage, setQueueMessage] = useState("")

  const {
    data: queueRes,
    isLoading: statsLoading,
    isRefetching: statsRefetching,
  } = useQueueStats({ refetchInterval: 10_000 })

  const stats = queueRes?.data ?? null
  const reanalyzeMutation = useReanalyzeAll()
  const retryMutation = useRetryDeadJobs()

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: queueKeys.all })
  }

  const handleReanalyze = async () => {
    setQueueMessage("")
    try {
      const { data } = await reanalyzeMutation.mutateAsync()
      setQueueMessage(
        `Queued ${data.queued} job(s)` +
          (data.dead_retried > 0 ? `, retried ${data.dead_retried} dead job(s)` : "")
      )
      handleRefresh()
    } catch (err) {
      setQueueMessage(err instanceof Error ? err.message : "Failed to trigger re-analysis")
    }
  }

  const handleRetryDead = async (type: string) => {
    setQueueMessage("")
    try {
      const { data } = await retryMutation.mutateAsync(type)
      setQueueMessage(`Retried ${data.count} dead ${type} job(s)`)
      handleRefresh()
    } catch (err) {
      setQueueMessage(err instanceof Error ? err.message : "Failed to retry dead jobs")
    }
  }

  const handleStatusClick = (type: QueueJobType, status: QueueJobStatus) => {
    jobsBrowserRef.current?.setFilter(type, status)
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
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleReanalyze}
            disabled={reanalyzeMutation.isPending}
          >
            {reanalyzeMutation.isPending ? (
              <Loader2 className="mr-2 size-4 animate-spin" />
            ) : (
              <Play className="mr-2 size-4" />
            )}
            {reanalyzeMutation.isPending ? "Queuing..." : "Re-analyze Unanalyzed Diffs"}
          </Button>
          {stats && stats.analyze.dead > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleRetryDead("analyze")}
              disabled={retryMutation.isPending}
            >
              {retryMutation.isPending ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RotateCcw className="mr-2 size-4" />
              )}
              Retry {stats.analyze.dead} Dead Job{stats.analyze.dead !== 1 ? "s" : ""}
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={statsRefetching}
          >
            <RefreshCw
              className={`mr-2 size-4 ${statsRefetching ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
        </div>
      </div>

      {/* Queue Stats Cards */}
      {statsLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : stats ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <QueueStatsCard
            title="Diff Queue"
            type="diff"
            stats={stats.diff}
            onStatusClick={handleStatusClick}
          />
          <QueueStatsCard
            title="Analyze Queue"
            type="analyze"
            stats={stats.analyze}
            onStatusClick={handleStatusClick}
          />
        </div>
      ) : (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Queue stats unavailable. Redis may not be running.
          </CardContent>
        </Card>
      )}

      {queueMessage && (
        <p className="text-sm text-muted-foreground">{queueMessage}</p>
      )}

      {/* Jobs Browser - replaces old Dead Jobs section */}
      <QueueJobsBrowser
        ref={jobsBrowserRef}
        diffStats={stats?.diff}
        analyzeStats={stats?.analyze}
      />
    </div>
  )
}
