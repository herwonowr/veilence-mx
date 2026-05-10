import { Badge } from "@/ui/components/badge"

const severityConfig: Record<string, { label: string; className: string }> = {
  critical: { label: "Critical", className: "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400" },
  high: { label: "High", className: "border-orange-500/30 bg-orange-500/10 text-orange-600 dark:text-orange-400" },
  medium: { label: "Medium", className: "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400" },
  low: { label: "Low", className: "border-muted-foreground/30 bg-muted/50 text-muted-foreground" },
}

interface SeverityBadgeProps {
  severity?: string | null
}

export const SeverityBadge = ({ severity }: SeverityBadgeProps) => {
  if (!severity) {
    return <span className="text-muted-foreground text-sm">-</span>
  }

  const config = severityConfig[severity.toLowerCase()]

  if (!config) {
    return <Badge variant="outline">{severity}</Badge>
  }

  return (
    <Badge variant="outline" className={config.className}>
      {config.label}
    </Badge>
  )
}
