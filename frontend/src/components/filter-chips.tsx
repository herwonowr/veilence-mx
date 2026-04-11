"use client"

import { X } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export interface ActiveFilter {
  /** Label shown on the chip, e.g. "Severity" */
  label: string
  /** Display value, e.g. "Critical" */
  value: string
  /** Called when the user dismisses this chip */
  onRemove: () => void
}

interface FilterChipsProps {
  filters: ActiveFilter[]
  /** Called when user clicks "Clear all" */
  onClearAll: () => void
}

export function FilterChips({ filters, onClearAll }: FilterChipsProps) {
  if (filters.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" role="list" aria-label="Active filters">
      {filters.map((filter) => (
        <Badge
          key={`${filter.label}:${filter.value}`}
          variant="secondary"
          className="gap-1 pl-2.5 pr-1.5 py-1"
          role="listitem"
        >
          <span className="text-muted-foreground">{filter.label}:</span>
          <span className="font-medium">{filter.value}</span>
          <button
            type="button"
            className="ml-1 rounded-full p-0.5 hover:bg-muted-foreground/20 transition-colors"
            onClick={filter.onRemove}
            aria-label={`Remove ${filter.label} filter`}
          >
            <X className="h-3 w-3" />
          </button>
        </Badge>
      ))}
      <Button
        variant="ghost"
        size="sm"
        className="h-7 text-xs text-muted-foreground"
        onClick={onClearAll}
        aria-label="Clear all filters"
      >
        Clear all
      </Button>
    </div>
  )
}
