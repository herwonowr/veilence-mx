"use client"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/ui"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/ui"
import { Input } from "@/ui"
import { Label } from "@/ui"
import { Textarea } from "@/ui"
import { useIsMobile } from "@/core"

export interface AuditLogDetailItem {
  id: string
  userId?: string
  userEmail?: string
  workspaceId?: string
  workspaceName?: string
  action: string
  resource: string
  resourceId: string
  details: string
  ipAddress: string
  userAgent: string
  correlationId: string
  createdAt: string
}

interface AuditLogDetailDialogProps {
  log: AuditLogDetailItem | null
  onOpenChange: (open: boolean) => void
}

const ReadonlyField = ({
  label,
  value,
  className,
}: {
  label: string
  value: string
  className?: string
}) => (
  <div className="space-y-1.5">
    <Label className="text-muted-foreground text-xs font-medium">{label}</Label>
    <Input
      readOnly
      value={value}
      className={`bg-muted/50 cursor-default focus-visible:ring-0 focus-visible:border-input ${className ?? ""}`}
    />
  </div>
)

const ReadonlyTextareaField = ({
  label,
  value,
  className,
}: {
  label: string
  value: string
  className?: string
}) => (
  <div className="space-y-1.5">
    <Label className="text-muted-foreground text-xs font-medium">{label}</Label>
    <Textarea
      readOnly
      value={value}
      className={`bg-muted/50 cursor-default resize-none focus-visible:ring-0 focus-visible:border-input ${className ?? ""}`}
      rows={3}
    />
  </div>
)

const AuditLogDetailContent = ({ log }: { log: AuditLogDetailItem }) => {
  const timestamp = new Date(log.createdAt).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
  })

  return (
    <div className="space-y-4">
      <ReadonlyField label="Timestamp" value={timestamp} />

      {log.userEmail && (
        <ReadonlyField label="User Email" value={log.userEmail} />
      )}

      {log.userId && !log.userEmail && (
        <ReadonlyField label="User ID" value={log.userId} className="font-mono text-xs" />
      )}

      {(log.workspaceName || log.workspaceId) && (
        <ReadonlyField label="Workspace" value={log.workspaceName || log.workspaceId || ""} />
      )}

      <ReadonlyField label="Action" value={log.action} />

      <ReadonlyField label="Resource" value={log.resource.replace(/_/g, " ")} />

      {log.resourceId && (
        <ReadonlyField label="Resource ID" value={log.resourceId} className="font-mono text-xs" />
      )}

      {log.details && (
        <ReadonlyTextareaField label="Details" value={log.details} />
      )}

      {log.ipAddress && (
        <ReadonlyField label="IP Address" value={log.ipAddress} className="font-mono text-xs" />
      )}

      {log.userAgent && (
        <ReadonlyTextareaField label="User Agent" value={log.userAgent} className="font-mono text-xs" />
      )}

      {log.correlationId && (
        <ReadonlyField label="Correlation ID" value={log.correlationId} className="font-mono text-xs" />
      )}
    </div>
  )
}

export const AuditLogDetailDialog = ({ log, onOpenChange }: AuditLogDetailDialogProps) => {
  const isMobile = useIsMobile()

  if (!log) return null

  if (isMobile) {
    return (
      <Sheet open={!!log} onOpenChange={onOpenChange}>
        <SheetContent side="bottom" className="max-h-[85vh] overflow-y-auto p-4">
          <SheetHeader>
            <SheetTitle>Audit Log Detail</SheetTitle>
            <SheetDescription>
              {log.action} - {log.resource}
            </SheetDescription>
          </SheetHeader>
          <div className="py-4">
            <AuditLogDetailContent log={log} />
          </div>
          <SheetFooter />
        </SheetContent>
      </Sheet>
    )
  }

  return (
    <Dialog open={!!log} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Audit Log Detail</DialogTitle>
          <DialogDescription>
            {log.action} - {log.resource}
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-[60vh] overflow-y-auto">
          <AuditLogDetailContent log={log} />
        </div>
        <DialogFooter showCloseButton />
      </DialogContent>
    </Dialog>
  )
}
