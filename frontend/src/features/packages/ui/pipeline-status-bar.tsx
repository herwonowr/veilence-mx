"use client"

import { Activity, Clock, Microscope, AlertCircle } from "lucide-react"
import { Badge } from "@/ui"
import { usePipelineStatus } from "@/features/packages/hooks/use-pipeline-status"
import Link from "next/link"
import { ROUTES } from "@/core"

export const PipelineStatusBar = () => {
  const { status, hasActiveJobs, isLoading } = usePipelineStatus()

  if (isLoading || !status || !hasActiveJobs) return null

  const segments: { label: string; count: number; icon: React.ReactNode; className: string }[] = [
    ...(status.pending > 0
      ? [{ label: "pending", count: status.pending, icon: <Clock className="size-3" />, className: "border-yellow-500/30 bg-yellow-500/10 text-yellow-700 dark:text-yellow-400" }]
      : []),
    ...(status.diffing > 0
      ? [{ label: "diffing", count: status.diffing, icon: <Activity className="size-3" />, className: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-400" }]
      : []),
    ...(status.analyzing > 0
      ? [{ label: "analyzing", count: status.analyzing, icon: <Microscope className="size-3" />, className: "border-purple-500/30 bg-purple-500/10 text-purple-700 dark:text-purple-400" }]
      : []),
    ...(status.error > 0
      ? [{ label: "failed", count: status.error, icon: <AlertCircle className="size-3" />, className: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400" }]
      : []),
  ]

  return (
    <Link
      href={ROUTES.SETTINGS_QUEUE}
      className="flex items-center gap-2 rounded-lg border bg-card px-4 py-2.5 text-sm text-card-foreground hover:bg-accent transition-colors"
    >
      <Activity className="size-4 shrink-0 animate-pulse" />
      <span className="font-medium">Pipeline active:</span>
      <div className="flex flex-wrap gap-1.5">
        {segments.map((seg) => (
          <Badge key={seg.label} variant="outline" className={seg.className}>
            {seg.icon}
            <span className="ml-1">{seg.count} {seg.label}</span>
          </Badge>
        ))}
      </div>
    </Link>
  )
}
