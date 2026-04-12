"use client"

import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

export interface SkeletonColumn {
  /** Width class for the skeleton bar (e.g., "w-24", "w-32") */
  width: string
  /** Optional header label — if provided, renders a real header */
  header?: string
}

interface TableSkeletonProps {
  /** Column configuration defining skeleton widths and optional headers */
  columns: SkeletonColumn[]
  /** Number of skeleton rows to render (default: 5) */
  rows?: number
  /** Whether to show the table header row (default: true) */
  showHeader?: boolean
}

/**
 * Skeleton loading state for tables. Renders pulse-animated skeleton bars
 * matching the real table's column structure.
 *
 * Usage:
 * ```tsx
 * <TableSkeleton
 *   columns={[
 *     { width: "w-32", header: "Name" },
 *     { width: "w-16", header: "Ecosystem" },
 *     { width: "w-20", header: "Version" },
 *   ]}
 *   rows={5}
 * />
 * ```
 */
export function TableSkeleton({
  columns,
  rows = 5,
  showHeader = true,
}: TableSkeletonProps) {
  return (
    <Table>
      {showHeader && (
        <TableHeader>
          <TableRow>
            {columns.map((col, i) => (
              <TableHead key={i}>
                {col.header ?? <Skeleton className="h-4 w-16" />}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
      )}
      <TableBody>
        {Array.from({ length: rows }).map((_, rowIdx) => (
          <TableRow key={rowIdx}>
            {columns.map((col, colIdx) => (
              <TableCell key={colIdx}>
                <Skeleton className={`h-4 ${col.width}`} />
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
