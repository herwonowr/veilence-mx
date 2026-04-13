"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import { Button } from "@/ui/components/button"
import { Textarea } from "@/ui/components/textarea"
import { Skeleton } from "@/ui/components/skeleton"
import { MessageSquare, Send, Loader2 } from "lucide-react"
import { useAlertNotes, useCreateAlertNote } from "@/features/alerts/hooks/use-alerts"
import { toAlertNoteViewModels } from "@/domains/alerts"

export const AlertNotesTimeline = ({ alertId }: { alertId: number }) => {
  const { data: notesRes, isLoading } = useAlertNotes(alertId)
  const createNoteMutation = useCreateAlertNote(alertId)
  const [noteContent, setNoteContent] = useState("")

  const notes = toAlertNoteViewModels(notesRes?.data ?? [])

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

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MessageSquare className="h-5 w-5" />
          Notes ({notes.length})
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
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

        {/* Timeline */}
        {isLoading ? (
          <div className="space-y-4">
            {Array.from({ length: 2 }).map((_, i) => (
              <div key={i} className="flex gap-4">
                <Skeleton className="h-[10px] w-[10px] rounded-full shrink-0 ml-[14px] mt-1.5" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-12 w-full" />
                </div>
              </div>
            ))}
          </div>
        ) : notes.length > 0 ? (
          <div className="relative space-y-0">
            {/* Timeline line */}
            <div className="absolute left-[19px] top-3 bottom-3 w-px bg-border" aria-hidden="true" />

            {notes.map((note) => (
              <div key={note.id} className="relative flex gap-4 pb-6 last:pb-0">
                {/* Timeline dot */}
                <div className="relative z-10 ml-[14px] mt-1.5 flex h-[10px] w-[10px] shrink-0 items-center justify-center rounded-full bg-muted-foreground ring-2 ring-background" />

                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium text-foreground">
                      {note.userEmail || `User #${note.userId}`}
                    </span>
                    <span className="text-xs text-muted-foreground" title={new Date(note.createdAt).toLocaleString()}>
                      {note.relativeTime}
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">
                    {note.content}
                  </p>
                </div>
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
  )
}
