"use client"

import { use, useState } from "react"
import Link from "next/link"
import { notFound } from "next/navigation"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"
import { ArrowLeft, MessageSquare, Send, Loader2, ExternalLink } from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { RequireOrg } from "@/components/require-org"
import { DetailError } from "@/components/detail-error"
import { useAlert, useUpdateAlert, useAlertNotes, useCreateAlertNote } from "@/features/alerts"
import type { AlertSeverity } from "@/types"

function severityVariant(s: AlertSeverity) {
  if (s === "critical") return "destructive" as const
  if (s === "high") return "destructive" as const
  if (s === "medium") return "default" as const
  return "secondary" as const
}

export default function AlertDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProtectedRoute>
      <RequireOrg feature="alert details">
        <AlertDetailContent params={params} />
      </RequireOrg>
    </ProtectedRoute>
  )
}

function AlertDetailContent({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const alertId = parseInt(id, 10)

  if (isNaN(alertId) || alertId <= 0) {
    notFound()
  }

  const { data: alertRes, isError, refetch } = useAlert(alertId)
  const updateMutation = useUpdateAlert()
  const { data: notesRes, isLoading: notesLoading } = useAlertNotes(alertId)
  const createNoteMutation = useCreateAlertNote(alertId)

  const [noteContent, setNoteContent] = useState("")

  const notes = notesRes?.data ?? []
  const alert = alertRes?.data ?? null

  const handleAddNote = async () => {
    if (!noteContent.trim()) return
    await createNoteMutation.mutateAsync(noteContent.trim())
    setNoteContent("")
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      handleAddNote()
    }
  }

  if (isError) return (
    <DetailError
      message="Failed to load alert details. The alert may not exist or the server is unavailable."
      onRetry={() => refetch()}
      backHref="/alerts"
      backLabel="Alerts"
    />
  )

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/alerts"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Alerts
        </Link>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h1 className="text-3xl font-bold">Alert #{alertId}</h1>
          {alert && (
            <div className="flex flex-wrap items-center gap-2">
              {alert.status === "new" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => updateMutation.mutate({ id: alertId, status: "acknowledged" })}
                >
                  Acknowledge
                </Button>
              )}
              {alert.status !== "resolved" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => updateMutation.mutate({ id: alertId, status: "resolved" })}
                >
                  Resolve
                </Button>
              )}
            </div>
          )}
        </div>
      </div>

      {alert ? (
        <Card>
          <CardContent className="pt-6 space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <Badge variant={severityVariant(alert.severity)} className="text-sm">
                {alert.severity}
              </Badge>
              <Badge variant="outline">{alert.status}</Badge>
              <span className="text-sm text-muted-foreground">
                {new Date(alert.createdAt).toLocaleString()}
              </span>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Package</p>
              <Link
                href={`/packages/${alert.packageId}`}
                className="text-primary hover:underline font-medium"
              >
                {alert.packageName}
                <span className="text-xs text-muted-foreground ml-1">({alert.packageRegistry})</span>
              </Link>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Message</p>
              <p className="text-sm leading-relaxed mt-1">{alert.message}</p>
            </div>
            <div>
              {(alert.releaseId ?? alert.analysisId) ? (
                <Link
                  href={`/releases/${alert.releaseId ?? alert.analysisId}`}
                  className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                >
                  <ExternalLink className="h-3.5 w-3.5" />
                  View Release Analysis
                </Link>
              ) : (
                <span className="inline-flex items-center gap-1 text-sm text-muted-foreground">
                  <ExternalLink className="h-3.5 w-3.5" />
                  No release linked
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="pt-6 space-y-4">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-4 w-64" />
            <Skeleton className="h-16 w-full" />
          </CardContent>
        </Card>
      )}

      {/* Notes / Comments */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            Notes ({notes.length})
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Add note form */}
          <div className="space-y-2">
            <Textarea
              placeholder="Add a note... (Ctrl+Enter to submit)"
              value={noteContent}
              onChange={(e) => setNoteContent(e.target.value)}
              onKeyDown={handleKeyDown}
              aria-label="Add a note to this alert"
            />
            <div className="flex justify-end">
              <Button
                size="sm"
                onClick={handleAddNote}
                disabled={!noteContent.trim() || createNoteMutation.isPending}
              >
                {createNoteMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Send className="mr-2 h-4 w-4" />
                )}
                Add Note
              </Button>
            </div>
          </div>

          {/* Notes list */}
          {notesLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : notes.length > 0 ? (
            <div className="space-y-3">
              {notes.map((note) => (
                <div key={note.id} className="rounded-lg border p-3 space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">
                      {note.userEmail || `User #${note.userId}`}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(note.createdAt).toLocaleString()}
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                    {note.content}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-center text-sm text-muted-foreground py-4">
              No notes yet. Add a note to track investigation progress.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
