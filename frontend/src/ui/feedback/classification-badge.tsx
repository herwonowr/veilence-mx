import { Badge } from "@/ui/components/badge"

export const CLASSIFICATIONS = [
  "malicious",
  "suspicious",
  "benign",
  "baseline",
] as const

export type ClassificationValue = (typeof CLASSIFICATIONS)[number]

const classificationVariant = (c: ClassificationValue) => {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  if (c === "benign") return "secondary" as const
  if (c === "baseline") return "outline" as const
  return "outline" as const
}

interface ClassificationBadgeProps {
  classification?: string | null
}

export const ClassificationBadge = ({ classification }: ClassificationBadgeProps) => {
  if (!classification) {
    return <span className="text-muted-foreground text-sm">-</span>
  }

  const normalized = classification.toLowerCase()
  const isKnown = CLASSIFICATIONS.includes(normalized as ClassificationValue)
  const variant = isKnown ? classificationVariant(normalized as ClassificationValue) : "outline"
  const label = classification.charAt(0).toUpperCase() + classification.slice(1)

  return (
    <Badge variant={variant}>
      {label}
    </Badge>
  )
}
