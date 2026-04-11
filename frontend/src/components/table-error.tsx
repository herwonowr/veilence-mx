"use client"

import { AlertTriangle, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableRow,
} from "@/components/ui/table"

interface TableErrorProps {
  /** Number of columns to span (for proper table layout) */
  colSpan: number
  /** Error message to display */
  message?: string
  /** Retry callback — renders a "Try again" button when provided */
  onRetry?: () => void
}

/**
 * Error state for tables. Displays an alert icon, message, and optional retry
 * button inside a table row that spans all columns.
 */
export function TableError({
  colSpan,
  message = "Failed to load data. Please try again.",
  onRetry,
}: TableErrorProps) {
  return (
    <Table>
      <TableBody>
        <TableRow>
          <TableCell colSpan={colSpan} className="text-center py-12">
            <div
              className="flex flex-col items-center gap-3"
              role="alert"
            >
              <AlertTriangle className="h-8 w-8 text-destructive" />
              <p className="text-sm text-muted-foreground">{message}</p>
              {onRetry && (
                <Button variant="outline" size="sm" onClick={onRetry}>
                  <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                  Try again
                </Button>
              )}
            </div>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  )
}
