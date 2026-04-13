import type { AlertNote } from "@/domains/alerts/types/alerts.types"

export interface AlertNoteViewModel {
  id: number
  alertId: number
  userId: number
  userEmail: string
  userInitials: string
  content: string
  createdAt: string
  updatedAt: string
  relativeTime: string
  isEdited: boolean
}

const getInitials = (email: string): string => {
  if (!email) return "?"
  const name = email.split("@")[0] ?? ""
  const parts = name.split(/[._-]/).filter(Boolean)
  if (parts.length >= 2) {
    return `${parts[0]![0]}${parts[1]![0]}`.toUpperCase()
  }
  return name.slice(0, 2).toUpperCase()
}

const formatRelativeTime = (dateStr: string): string => {
  const now = Date.now()
  const date = new Date(dateStr).getTime()
  const diffMs = now - date

  if (diffMs < 0) return "just now"

  const minutes = Math.floor(diffMs / 60_000)
  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes}m ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`

  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`

  return new Date(dateStr).toLocaleDateString()
}

export const toAlertNoteViewModel = (note: AlertNote): AlertNoteViewModel => ({
  id: note.id,
  alertId: note.alertId,
  userId: note.userId,
  userEmail: note.userEmail,
  userInitials: getInitials(note.userEmail),
  content: note.content,
  createdAt: note.createdAt,
  updatedAt: note.updatedAt,
  relativeTime: formatRelativeTime(note.createdAt),
  isEdited: new Date(note.updatedAt).getTime() - new Date(note.createdAt).getTime() > 1000,
})

export const toAlertNoteViewModels = (notes: AlertNote[]): AlertNoteViewModel[] =>
  notes.map(toAlertNoteViewModel)
