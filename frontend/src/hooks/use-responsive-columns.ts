import { useMemo } from "react"
import { useBreakpoint, type Breakpoint } from "@/hooks/use-breakpoint"
import type { VisibilityState } from "@tanstack/react-table"

/**
 * Column visibility spec: which columns to hide at each breakpoint.
 * Keys are column accessorKey or id values.
 * Value is the minimum breakpoint at which the column is visible.
 *
 * Example: { rank: "desktop", isCustom: "desktop", latestVersion: "tablet" }
 * - rank: hidden on mobile + tablet, visible on desktop + xl
 * - latestVersion: hidden on mobile, visible on tablet + desktop + xl
 */
export type ColumnBreakpoints = Record<string, Breakpoint>

const BREAKPOINT_ORDER: Breakpoint[] = ["mobile", "tablet", "desktop", "xl"]

function isAtLeast(current: Breakpoint, minimum: Breakpoint): boolean {
  return BREAKPOINT_ORDER.indexOf(current) >= BREAKPOINT_ORDER.indexOf(minimum)
}

export function useResponsiveColumns(columnBreakpoints: ColumnBreakpoints): VisibilityState {
  const breakpoint = useBreakpoint()

  return useMemo(() => {
    const visibility: VisibilityState = {}
    for (const [columnId, minBreakpoint] of Object.entries(columnBreakpoints)) {
      visibility[columnId] = isAtLeast(breakpoint, minBreakpoint)
    }
    return visibility
  }, [breakpoint, columnBreakpoints])
}
