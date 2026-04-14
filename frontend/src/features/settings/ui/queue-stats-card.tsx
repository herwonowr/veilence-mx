"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/ui/components/tooltip"
import {
  Clock,
  Activity,
  CheckCircle2,
  XCircle,
  Skull,
  ListOrdered,
  ChevronRight,
  Info,
} from "lucide-react"
import type { QueueStats } from "@/domains/queue"
import type { QueueJobStatus, QueueJobType } from "@/domains/queue"

interface QueueStatsCardProps {
  title: string
  type: QueueJobType
  stats: QueueStats
  onStatusClick: (type: QueueJobType, status: QueueJobStatus) => void
}

export const QueueStatsCard = ({
  title,
  type,
  stats,
  onStatusClick,
}: QueueStatsCardProps) => {
  const total =
    stats.pending + stats.processing + stats.completed + stats.failed + stats.dead

  const clickableItems: {
    label: string
    value: number
    icon: React.ReactNode
    color: string
    status: QueueJobStatus
  }[] = [
    {
      label: "Pending",
      value: stats.pending,
      icon: <Clock className="size-4 text-muted-foreground" />,
      color: "text-muted-foreground",
      status: "pending",
    },
    {
      label: "Processing",
      value: stats.processing,
      icon: <Activity className="size-4 text-blue-500" />,
      color: "text-blue-500",
      status: "processing",
    },
  ]

  const staticItems: {
    label: string
    value: number
    icon: React.ReactNode
    color: string
    tooltip: string
  }[] = [
    {
      label: "Completed",
      value: stats.completed,
      icon: <CheckCircle2 className="size-4 text-green-500" />,
      color: "text-green-500",
      tooltip: "Total completed jobs (counter only \u2014 individual jobs expire after 1 hour)",
    },
    {
      label: "Total Dead",
      value: stats.failed,
      icon: <XCircle className="size-4 text-orange-500" />,
      color: "text-orange-500",
      tooltip: "Cumulative count of jobs that exceeded max retries (includes retried jobs)",
    },
  ]

  const deadItem = {
    label: "Dead",
    value: stats.dead,
    icon: <Skull className="size-4 text-destructive" />,
    color: "text-destructive",
    status: "dead" as QueueJobStatus,
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <ListOrdered className="size-4" />
          {title}
        </CardTitle>
        <p className="text-xs text-muted-foreground" aria-live="polite">
          {total} total jobs
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-1">
          {/* Clickable: Pending, Processing */}
          {clickableItems.map((item) => (
            <button
              key={item.label}
              type="button"
              className="flex w-full items-center justify-between rounded-md px-2 py-1.5 transition-colors duration-150 hover:bg-muted/50 cursor-pointer"
              onClick={() => onStatusClick(type, item.status)}
              aria-label={`View ${item.value} ${item.label.toLowerCase()} ${type} jobs`}
            >
              <div className="flex items-center gap-2 text-sm">
                {item.icon}
                <span>{item.label}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className={`text-lg font-semibold tabular-nums ${item.color}`}>
                  {item.value.toLocaleString()}
                </span>
                <ChevronRight className={`size-4 text-muted-foreground/50 ${item.value === 0 ? "opacity-40" : ""}`} />
              </div>
            </button>
          ))}

          {/* Static: Completed, Total Dead */}
          {staticItems.map((item) => (
            <div
              key={item.label}
              className="flex items-center justify-between px-2 py-1.5"
            >
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                {item.icon}
                <span>{item.label}</span>
                <Tooltip>
                  <TooltipTrigger
                    render={<button type="button" className="inline-flex" />}
                  >
                    <Info className="size-3 text-muted-foreground/60" />
                  </TooltipTrigger>
                  <TooltipContent>
                    {item.tooltip}
                  </TooltipContent>
                </Tooltip>
              </div>
              <span className={`text-lg font-semibold tabular-nums ${item.color}`}>
                {item.value.toLocaleString()}
              </span>
            </div>
          ))}

          {/* Clickable: Dead */}
          <button
            type="button"
            className="flex w-full items-center justify-between rounded-md px-2 py-1.5 transition-colors duration-150 hover:bg-muted/50 cursor-pointer"
            onClick={() => onStatusClick(type, deadItem.status)}
            aria-label={`View ${deadItem.value} ${deadItem.label.toLowerCase()} ${type} jobs`}
          >
            <div className="flex items-center gap-2 text-sm">
              {deadItem.icon}
              <span>{deadItem.label}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`text-lg font-semibold tabular-nums ${deadItem.color}`}>
                {deadItem.value.toLocaleString()}
              </span>
              <ChevronRight className={`size-4 text-muted-foreground/50 ${deadItem.value === 0 ? "opacity-40" : ""}`} />
            </div>
          </button>
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
