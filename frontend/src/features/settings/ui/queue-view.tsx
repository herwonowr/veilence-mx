"use client"

import { useRef } from "react"
import { Card, CardContent } from "@/ui/components/card"
import { Button } from "@/ui/components/button"
import { RefreshCw, Loader2 } from "lucide-react"
import { useQueueStats, queueKeys } from "@/features/settings/hooks/use-queue"
import { useQueryClient } from "@tanstack/react-query"
import { QueueStatsCard } from "@/features/settings/ui/queue-stats-card"
import { QueueJobsBrowser, type QueueJobsBrowserHandle } from "@/features/settings/ui/queue-jobs-browser"
import type { QueueJobStatus, QueueJobType } from "@/domains/queue"

export const QueueView = () => {
  const jobsBrowserRef = useRef<QueueJobsBrowserHandle>(null)
  const queryClient = useQueryClient()

  const {
    data: queueRes,
    isLoading: statsLoading,
    isRefetching: statsRefetching,
  } = useQueueStats({ refetchInterval: 10_000 })

  const stats = queueRes?.data ?? null

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: queueKeys.all })
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

      {/* Jobs Browser - replaces old Dead Jobs section */}
      <QueueJobsBrowser
        ref={jobsBrowserRef}
        diffStats={stats?.diff}
        analyzeStats={stats?.analyze}
      />
    </div>
  )
}
