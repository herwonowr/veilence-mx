import { Badge } from "@/ui/components/badge"

export const RELEASE_STATUSES = [
  "pending",
  "diffing",
  "analyzing",
  "completed",
  "error",
  "in_progress",
] as const

export type ReleaseStatusValue = (typeof RELEASE_STATUSES)[number]

const statusConfig: Record<ReleaseStatusValue, { label: string; className: string }> = {
  pending: {
    label: "Pending",
    className: "border-muted-foreground/30 bg-muted/50 text-muted-foreground",
  },
  diffing: {
    label: "Diffing",
    className: "border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400",
  },
  analyzing: {
    label: "Analyzing",
    className: "border-purple-500/30 bg-purple-500/10 text-purple-600 dark:text-purple-400",
  },
  completed: {
    label: "Completed",
    className: "border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400",
  },
  error: {
    label: "Error",
    className: "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400",
  },
  in_progress: {
    label: "In Progress",
    className: "border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400",
  },
}

interface ReleaseStatusBadgeProps {
  status: ReleaseStatusValue
}

export const ReleaseStatusBadge = ({ status }: ReleaseStatusBadgeProps) => {
  const config = statusConfig[status]
  if (!config) {
    return <Badge variant="outline">{status}</Badge>
  }
  return (
    <Badge variant="outline" className={config.className}>
      {config.label}
    </Badge>
  )
}
