"use client"

import { useState, useEffect } from "react"
import { Badge } from "@/ui/components/badge"
import { Button } from "@/ui/components/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/ui/components/dialog"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/ui/components/sheet"
import { Loader2 } from "lucide-react"
import { useIsMobile } from "@/core"
import type { QueueJob } from "@/domains/queue"

interface QueueJobDetailDialogProps {
  job: QueueJob | null
  onOpenChange: (open: boolean) => void
  onRetry: (jobId: string) => void
  isRetrying?: boolean
}

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
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
  })
}

const JobDetailContent = ({
  job,
  onRetry,
  isRetrying,
}: {
  job: QueueJob
  onRetry: (jobId: string) => void
  isRetrying?: boolean
}) => {
  const nowSeconds = useNowSeconds()
  const isRetryingJob = job.status === "pending" && !!job.lastError
  const isProcessing = job.status === "processing"
  const isDead = job.status === "dead"
  const duration = isProcessing
    ? Math.max(0, nowSeconds - job.updatedAt)
    : 0
  const isPossiblyStuck = isProcessing && duration > STUCK_THRESHOLD_SECONDS

  return (
    <>
      <dl className="space-y-3 text-sm">
        {/* Status */}
        <div className="flex items-center justify-between">
          <dt className="text-muted-foreground">Status</dt>
          <dd className="flex items-center gap-2">
            {job.status === "pending" && (
              <Badge variant="outline">Pending</Badge>
            )}
            {job.status === "processing" && (
              <Badge
                variant="outline"
                className="border-blue-500/30 bg-blue-500/10 text-blue-600"
              >
                Processing
              </Badge>
            )}
            {job.status === "dead" && (
              <Badge variant="destructive">Dead</Badge>
            )}
            {isRetryingJob && (
              <Badge
                variant="outline"
                className="border-orange-500/30 bg-orange-500/10 text-orange-600"
              >
                Retrying
              </Badge>
            )}
            {isPossiblyStuck && (
              <Badge
                variant="outline"
                className="animate-pulse border-orange-500/30 bg-orange-500/10 text-orange-600"
              >
                Possibly Stuck
              </Badge>
            )}
          </dd>
        </div>

        {/* Reference */}
        <div className="flex items-center justify-between">
          <dt className="text-muted-foreground">Reference</dt>
          <dd className="font-mono text-xs">#{job.referenceId}</dd>
        </div>

        {/* Type */}
        <div className="flex items-center justify-between">
          <dt className="text-muted-foreground">Type</dt>
          <dd>
            <Badge variant="outline" className="capitalize">
              {job.type}
            </Badge>
          </dd>
        </div>

        {/* Attempts */}
        <div className="flex items-center justify-between">
          <dt className="text-muted-foreground">Attempts</dt>
          <dd>
            {job.attempts} / {job.maxAttempts}
          </dd>
        </div>

        {/* Created */}
        <div className="flex items-center justify-between">
          <dt className="text-muted-foreground">Created</dt>
          <dd className="text-xs">{formatTimestamp(job.createdAt)}</dd>
        </div>

        {/* Updated */}
        <div className="flex items-center justify-between">
          <dt className="text-muted-foreground">Updated</dt>
          <dd className="text-xs">{formatTimestamp(job.updatedAt)}</dd>
        </div>

        {/* Next Run — only for retrying pending jobs */}
        {isRetryingJob && job.nextRunAt > 0 && (
          <div className="flex items-center justify-between">
            <dt className="text-muted-foreground">Next Run</dt>
            <dd className="text-xs">{formatTimestamp(job.nextRunAt)}</dd>
          </div>
        )}

        {/* Duration — only for processing jobs */}
        {isProcessing && (
          <div className="flex items-center justify-between">
            <dt className="text-muted-foreground">Duration</dt>
            <dd className={isPossiblyStuck ? "text-orange-600 font-medium" : ""}>
              {formatDuration(duration)}
            </dd>
          </div>
        )}
      </dl>

      {/* Last Error */}
      {job.lastError && (
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">Last Error</p>
          <pre
            className="max-h-60 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted p-3 text-xs text-destructive"
            role="log"
            aria-label="Error message"
          >
            {job.lastError}
          </pre>
        </div>
      )}

      {/* Footer actions */}
      {isDead && (
        <div className="flex justify-end gap-2">
          <Button
            onClick={() => onRetry(job.id)}
            disabled={isRetrying}
            aria-label={`Retry job ${job.id}`}
          >
            {isRetrying && <Loader2 className="mr-2 size-4 animate-spin" />}
            Retry
          </Button>
        </div>
      )}
    </>
  )
}

export const QueueJobDetailDialog = ({
  job,
  onOpenChange,
  onRetry,
  isRetrying,
}: QueueJobDetailDialogProps) => {
  const isMobile = useIsMobile()

  if (!job) return null

  if (isMobile) {
    return (
      <Sheet open={!!job} onOpenChange={onOpenChange}>
        <SheetContent side="bottom" className="max-h-[85vh] overflow-y-auto p-4">
          <SheetHeader>
            <SheetTitle>
              Job #{job.id} - {job.type}
            </SheetTitle>
            <SheetDescription>Job detail information</SheetDescription>
          </SheetHeader>
          <div className="space-y-4 py-4">
            <JobDetailContent
              job={job}
              onRetry={(id) => {
                onRetry(id)
                onOpenChange(false)
              }}
              isRetrying={isRetrying}
            />
          </div>
          <SheetFooter />
        </SheetContent>
      </Sheet>
    )
  }

  return (
    <Dialog open={!!job} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            Job #{job.id} - {job.type}
          </DialogTitle>
          <DialogDescription>Job detail information</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <JobDetailContent
            job={job}
            onRetry={(id) => {
              onRetry(id)
              onOpenChange(false)
            }}
            isRetrying={isRetrying}
          />
        </div>
        <DialogFooter showCloseButton />
      </DialogContent>
    </Dialog>
  )
}
