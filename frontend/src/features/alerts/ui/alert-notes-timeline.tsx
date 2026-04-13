"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import { Button, buttonVariants } from "@/ui/components/button"
import { Textarea } from "@/ui/components/textarea"
import { Skeleton } from "@/ui/components/skeleton"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/components/dropdown-menu"
import { ConfirmDialog } from "@/ui/feedback/confirm-dialog"
import { MessageSquare, Send, Loader2, MoreHorizontal, Pencil, Trash2 } from "lucide-react"
import { useAuth } from "@/core/providers/auth-provider"
import {
  useAlertNotes,
  useCreateAlertNote,
  useUpdateAlertNote,
  useDeleteAlertNote,
} from "@/features/alerts/hooks/use-alerts"
import { toAlertNoteViewModels } from "@/domains/alerts"
import type { AlertNoteViewModel } from "@/domains/alerts"

// ─── Helpers ──────────────────────────────────────────────────────────

const truncate = (str: string, maxLen: number): string =>
  str.length <= maxLen ? str : `${str.slice(0, maxLen)}…`

// ─── Individual Note Item ──────────────────────────────────────────────

interface NoteItemProps {
  note: AlertNoteViewModel
  isOwner: boolean
  isEditing: boolean
  editContent: string
  onStartEdit: () => void
  onCancelEdit: () => void
  onSaveEdit: () => void
  onEditContentChange: (value: string) => void
  isSaving: boolean
  onDelete: () => void
}

const NoteItem = ({
  note,
  isOwner,
  isEditing,
  editContent,
  onStartEdit,
  onCancelEdit,
  onSaveEdit,
  onEditContentChange,
  isSaving,
  onDelete,
}: NoteItemProps) => {
  const canSave =
    editContent.trim().length > 0 &&
    editContent.trim() !== note.content &&
    !isSaving

  const handleEditKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      if (canSave) onSaveEdit()
    }
    if (e.key === "Escape") {
      e.preventDefault()
      onCancelEdit()
    }
  }

  return (
    <div className="relative flex gap-4 pb-6 last:pb-0">
      {/* Timeline dot */}
      <div className="relative z-10 ml-[14px] mt-1.5 flex h-[10px] w-[10px] shrink-0 items-center justify-center rounded-full bg-muted-foreground ring-2 ring-background" />

      <div className="flex-1 min-w-0">
        {/* Header row */}
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-foreground">
            {note.userEmail || `User #${note.userId}`}
          </span>
          <span
            className="text-xs text-muted-foreground"
            title={new Date(note.createdAt).toLocaleString()}
          >
            {note.relativeTime}
          </span>
          {note.isEdited && (
            <span
              className="text-xs text-muted-foreground"
              title={`Edited ${new Date(note.updatedAt).toLocaleString()}`}
            >
              (edited)
            </span>
          )}

          {/* Owner-only action menu — pushed to end */}
          {isOwner && !isEditing && (
            <div className="ml-auto">
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <button
                      className={buttonVariants({ variant: "ghost", size: "icon-xs" })}
                      aria-label="Note actions"
                    />
                  }
                >
                  <MoreHorizontal className="h-3.5 w-3.5" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" side="bottom" sideOffset={4}>
                  <DropdownMenuItem onClick={onStartEdit}>
                    <Pencil /> Edit
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem variant="destructive" onClick={onDelete}>
                    <Trash2 /> Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )}
        </div>

        {/* Content — switches between read and edit */}
        {isEditing ? (
          <div className="mt-1">
            <Textarea
              value={editContent}
              onChange={(e) => onEditContentChange(e.target.value)}
              onKeyDown={handleEditKeyDown}
              autoFocus
              aria-label="Edit note content"
              placeholder="Ctrl+Enter to save, Escape to cancel"
            />
            <div className="flex justify-end gap-2 mt-2">
              <Button variant="outline" size="sm" onClick={onCancelEdit} disabled={isSaving}>
                Cancel
              </Button>
              <Button size="sm" onClick={onSaveEdit} disabled={!canSave}>
                {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Save
              </Button>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">
            {note.content}
          </p>
        )}
      </div>
    </div>
  )
}

// ─── Notes Timeline ───────────────────────────────────────────────────

export const AlertNotesTimeline = ({ alertId }: { alertId: number }) => {
  const { user } = useAuth()
  const { data: notesRes, isLoading } = useAlertNotes(alertId)
  const createNoteMutation = useCreateAlertNote(alertId)
  const updateNoteMutation = useUpdateAlertNote(alertId)
  const deleteNoteMutation = useDeleteAlertNote(alertId)

  const [noteContent, setNoteContent] = useState("")

  // Edit state — only one note can be edited at a time
  const [editingNoteId, setEditingNoteId] = useState<number | null>(null)
  const [editContent, setEditContent] = useState("")

  // Delete state — controlled ConfirmDialog
  const [deletingNote, setDeletingNote] = useState<AlertNoteViewModel | null>(null)

  // Backend returns newest-first (DESC) — render in array order
  const notes = toAlertNoteViewModels(notesRes?.data ?? [])

  // ─── Create ───────────────────────────────────────────────

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

  // ─── Edit ─────────────────────────────────────────────────

  const handleStartEdit = (note: AlertNoteViewModel) => {
    setEditingNoteId(note.id)
    setEditContent(note.content)
  }

  const handleCancelEdit = () => {
    setEditingNoteId(null)
    setEditContent("")
  }

  const handleSaveEdit = async () => {
    if (editingNoteId === null) return
    const trimmed = editContent.trim()
    if (!trimmed) return
    await updateNoteMutation.mutateAsync({ noteId: editingNoteId, content: trimmed })
    setEditingNoteId(null)
    setEditContent("")
  }

  // ─── Delete ───────────────────────────────────────────────

  const handleConfirmDelete = async () => {
    if (!deletingNote) return
    await deleteNoteMutation.mutateAsync(deletingNote.id)
    setDeletingNote(null)
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
              <NoteItem
                key={note.id}
                note={note}
                isOwner={user?.id === note.userId}
                isEditing={editingNoteId === note.id}
                editContent={editingNoteId === note.id ? editContent : note.content}
                onStartEdit={() => handleStartEdit(note)}
                onCancelEdit={handleCancelEdit}
                onSaveEdit={handleSaveEdit}
                onEditContentChange={setEditContent}
                isSaving={updateNoteMutation.isPending}
                onDelete={() => setDeletingNote(note)}
              />
            ))}
          </div>
        ) : (
          <p className="text-center text-sm text-muted-foreground py-4">
            No notes yet. Add a note to track investigation progress.
          </p>
        )}

        {/* Delete confirmation dialog — controlled mode */}
        <ConfirmDialog
          open={deletingNote !== null}
          onOpenChange={(open) => {
            if (!open) setDeletingNote(null)
          }}
          title="Delete note"
          description="This note will be permanently removed. This action cannot be undone."
          actionLabel="Delete"
          details={
            deletingNote
              ? [
                  { label: "Content", value: truncate(deletingNote.content, 80) },
                  { label: "Author", value: deletingNote.userEmail },
                ]
              : []
          }
          onConfirm={handleConfirmDelete}
        />
      </CardContent>
    </Card>
  )
}
