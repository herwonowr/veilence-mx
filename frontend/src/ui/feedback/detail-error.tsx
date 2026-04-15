"use client"

import Link from "next/link"
import { AlertTriangle, RefreshCw, ArrowLeft } from "lucide-react"
import { Button } from "@/ui/components/button"
import { Card, CardContent } from "@/ui/components/card"

interface DetailErrorProps {
  /** Error message to display */
  message?: string
  /** Retry callback - renders a "Try again" button when provided */
  onRetry?: () => void
  /** Back link URL (e.g. "/packages") */
  backHref?: string
  /** Back link label (e.g. "Packages") */
  backLabel?: string
}

/**
 * Error state for detail pages. Displays an alert icon, message,
 * optional retry button, and optional "go back to list" link.
 */
export const DetailError = ({
  message = "Failed to load data. Please try again.",
  onRetry,
  backHref,
  backLabel = "Go back",
}: DetailErrorProps) => (
  <div className="space-y-6 p-8">
    {backHref && (
      <Link
        href={backHref}
        className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        {backLabel}
      </Link>
    )}
    <Card>
      <CardContent className="py-12">
        <div
          className="flex flex-col items-center gap-3"
          role="alert"
        >
          <AlertTriangle className="h-8 w-8 text-destructive" />
          <p className="text-sm text-muted-foreground">{message}</p>
          <div className="flex items-center gap-3">
            {onRetry && (
              <Button variant="outline" size="sm" onClick={onRetry}>
                <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                Try again
              </Button>
            )}
            {backHref && (
              <Link
                href={backHref}
                className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
              >
                <ArrowLeft className="h-3.5 w-3.5" />
                {backLabel}
              </Link>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
)
