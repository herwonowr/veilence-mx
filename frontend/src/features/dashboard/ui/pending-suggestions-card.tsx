"use client"

import Link from "next/link"
import { Card, CardContent, Badge, Button } from "@/ui"
import { Radar } from "lucide-react"
import { usePendingSuggestionsCount } from "@/features/dashboard/hooks/use-pending-suggestions"

export const PendingSuggestionsCard = () => {
  const { data: count, isLoading } = usePendingSuggestionsCount()

  if (isLoading || !count || count === 0) return null

  return (
    <Card className="border-amber-500/40">
      <CardContent className="flex items-center gap-4 py-4">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-amber-500/10">
          <Radar className="h-5 w-5 text-amber-500" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-0.5">
            <p className="text-sm font-medium">Package Suggestions Pending Review</p>
            <Badge variant="secondary">{count}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Discovered packages need your approval before monitoring begins.
          </p>
        </div>
        <Link href="/packages/suggestions" className="shrink-0">
          <Button size="sm" variant="outline">
            <span className="relative mr-1.5 flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-500 opacity-75" />
              <span className="relative inline-flex h-2 w-2 rounded-full bg-amber-500" />
            </span>
            Review Suggestions
          </Button>
        </Link>
      </CardContent>
    </Card>
  )
}
