import { cn } from "@/lib/utils"

interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: string
  children?: React.ReactNode
  className?: string
}

/**
 * EmptyState — a reusable component for list pages with no data.
 * Renders an icon, title, optional description, and optional action buttons.
 */
export function EmptyState({
  icon,
  title,
  description,
  children,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 py-12 text-center",
        className
      )}
    >
      {icon && (
        <div className="flex items-center justify-center text-muted-foreground">
          {icon}
        </div>
      )}
      <p className="text-sm font-medium text-muted-foreground">{title}</p>
      {description && (
        <p className="max-w-sm text-xs text-muted-foreground">{description}</p>
      )}
      {children && <div className="flex flex-wrap items-center gap-2 mt-1">{children}</div>}
    </div>
  )
}
