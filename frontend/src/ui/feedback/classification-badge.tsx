import { Badge } from "@/ui/components/badge"

const classificationConfig: Record<string, { label: string; className: string }> = {
  malicious: { label: "Malicious", className: "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400" },
  suspicious: { label: "Suspicious", className: "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400" },
  benign: { label: "Benign", className: "border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400" },
  baseline: { label: "Baseline", className: "border-muted-foreground/30 bg-muted/50 text-muted-foreground" },
}

interface ClassificationBadgeProps {
  classification?: string | null
}

export const ClassificationBadge = ({ classification }: ClassificationBadgeProps) => {
  if (!classification) {
    return <span className="text-muted-foreground text-sm">-</span>
  }

  const config = classificationConfig[classification.toLowerCase()]

  if (!config) {
    return <Badge variant="outline">{classification}</Badge>
  }

  return (
    <Badge variant="outline" className={config.className}>
      {config.label}
    </Badge>
  )
}
