"use client"

import { Button, Card, CardContent } from "@/ui"
import { AlertTriangle } from "lucide-react"

const GlobalError = ({
  error,
  unstable_retry,
}: {
  error: Error & { digest?: string }
  unstable_retry: () => void
}) => {
  return (
    <html>
      <body className="min-h-screen flex items-center justify-center bg-background text-foreground p-4">
        <Card className="max-w-md w-full">
          <CardContent className="pt-6 text-center space-y-4">
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/20">
              <AlertTriangle className="size-6 text-red-600" />
            </div>
            <h2 className="text-xl font-semibold">Something went wrong</h2>
            <p className="text-sm text-muted-foreground">
              A critical error occurred. Please try reloading the page.
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
      </body>
    </html>
  )
}
export default GlobalError
