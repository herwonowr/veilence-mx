"use client"

import { useEffect } from "react"
import { Button } from "@/ui/components/button"
import { Card, CardContent } from "@/ui/components/card"
import { AlertTriangle } from "lucide-react"

const ErrorPage = ({
  error,
  unstable_retry,
}: {
  error: Error & { digest?: string }
  unstable_retry: () => void
}) => {
  useEffect(() => {
    console.error("Runtime error:", error)
  }, [error])

  return (
    <div className="flex min-h-[50vh] items-center justify-center p-4">
      <Card className="max-w-md w-full">
        <CardContent className="pt-6 text-center space-y-4">
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-destructive/10">
            <AlertTriangle className="size-6 text-destructive" />
          </div>
          <h2 className="text-xl font-semibold">Something went wrong</h2>
          <p className="text-sm text-muted-foreground">
            An unexpected error occurred. Please try again or contact support if the problem persists.
          </p>
          {error.digest && (
            <p className="text-xs text-muted-foreground font-mono">
              Error ID: {error.digest}
            </p>
          )}
          <Button onClick={() => unstable_retry()} className="w-full">
            Try again
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
export default ErrorPage
