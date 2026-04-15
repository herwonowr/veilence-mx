"use client"

import { useState } from "react"
import { Button } from "@/ui/components/button"
import {
  Card,
  CardContent,
  CardDescription,
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
import { TableSkeleton, type SkeletonColumn } from "@/ui/feedback/table-skeleton"
import { TableError } from "@/ui/feedback/table-error"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { ConfirmDialog, type ConfirmDialogDetail } from "@/ui/feedback/confirm-dialog"
import { Monitor, Trash2, ShieldCheck } from "lucide-react"
import { Badge } from "@/ui/components/badge"
import { useSessions, useRevokeSession } from "@/features/account/hooks/use-sessions"

const parseUserAgent = (ua: string): string => {
  if (ua.includes("Chrome") && !ua.includes("Edg")) return "Chrome"
  if (ua.includes("Edg")) return "Edge"
  if (ua.includes("Firefox")) return "Firefox"
  if (ua.includes("Safari") && !ua.includes("Chrome")) return "Safari"
  if (ua.includes("curl")) return "curl"
  if (ua.length > 60) return ua.slice(0, 60) + "..."
  return ua || "Unknown"
}

export const SessionsView = () => {
  const { data: sessionsRes, isLoading, isError, refetch } = useSessions()

  // Revoke confirmation state
  const [revokeTarget, setRevokeTarget] = useState<{
    id: number
    details: ConfirmDialogDetail[]
  } | null>(null)

  // The current session's delete button is disabled in the UI, so a successful
  // revoke is always for a non-current session. No token refresh needed - the
  // previous approach called apiRefreshToken to "test" if the current session
  // was still alive, but that consumed the refresh token via rotation without
  // storing the new one, causing a race condition that logged the user out.
  const revokeMutation = useRevokeSession()

  const sessionsSkeletonColumns: SkeletonColumn[] = [
    { width: "w-20", header: "Browser / Client" },
    { width: "w-24", header: "IP Address" },
    { width: "w-24", header: "Created" },
    { width: "w-24", header: "Last Active" },
    { width: "w-20", header: "Expires" },
    { width: "w-8", header: "" },
  ]

  const sessions = sessionsRes?.data ?? []

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Active Sessions</h1>
        <p className="text-sm text-muted-foreground">
          View and manage your active sessions across devices
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Monitor className="size-5" />
            Sessions
          </CardTitle>
          <CardDescription>
            These are the devices and browsers currently signed in to your
            account. Revoke any session you don&apos;t recognize.
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <TableSkeleton columns={sessionsSkeletonColumns} rows={5} />
          ) : isError ? (
            <TableError colSpan={6} onRetry={() => refetch()} />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Browser / Client</TableHead>
                  <TableHead>IP Address</TableHead>
                  <TableHead className="hidden md:table-cell">Created</TableHead>
                  <TableHead>Last Active</TableHead>
                  <TableHead className="hidden lg:table-cell">Expires</TableHead>
                  <TableHead className="w-[1%] whitespace-nowrap text-right">
                    <span className="sr-only">Actions</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((session) => (
                  <TableRow
                    key={session.id}
                    className={session.isCurrent ? "bg-primary/5" : undefined}
                    aria-label={session.isCurrent ? "Current session" : undefined}
                  >
                    <TableCell className="text-sm font-medium">
                      <span className="flex items-center gap-2">
                        {parseUserAgent(session.userAgent)}
                        {session.isCurrent && (
                          <Badge variant="secondary" className="gap-1">
                            <ShieldCheck className="size-3" />
                            Current session
                          </Badge>
                        )}
                      </span>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {session.ipAddress}
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
                      {new Date(session.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(session.lastActive).toLocaleString()}
                    </TableCell>
                    <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                      {new Date(session.expiresAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      {session.isCurrent ? (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          disabled
                          aria-label="Cannot revoke current session"
                        >
                          <Trash2 className="size-4 text-muted-foreground/40" />
                        </Button>
                      ) : (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() =>
                            setRevokeTarget({
                              id: session.id,
                              details: [
                                { label: "Device", value: parseUserAgent(session.userAgent) },
                                { label: "IP Address", value: session.ipAddress },
                                { label: "Last Active", value: new Date(session.lastActive).toLocaleString() },
                              ],
                            })
                          }
                          disabled={revokeMutation.isPending}
                          aria-label={`Revoke session for ${parseUserAgent(session.userAgent)}`}
                        >
                          <Trash2 className="size-4 text-destructive" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {sessions.length === 0 && (
                  <TableEmptyState
                    colSpan={6}
                    icon={<Monitor className="h-8 w-8" />}
                    title="No active sessions found."
                    description="Your session information will appear here when you sign in on other devices."
                  />
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Revoke Session Confirmation */}
      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(open) => { if (!open) setRevokeTarget(null) }}
        title="Revoke Session?"
        description="Are you sure you want to revoke this session? The device will be signed out immediately."
        details={revokeTarget?.details}
        actionLabel="Revoke"
        onConfirm={async () => {
          if (revokeTarget) {
            await revokeMutation.mutateAsync(revokeTarget.id)
          }
        }}
      />
    </div>
  )
}

