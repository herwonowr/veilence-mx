"use client"

import { PartyPopper, Loader2 } from "lucide-react"
import { Button } from "@/ui"

interface CompleteStepProps {
  isSubmitting: boolean
  onComplete: () => void
  onBack: () => void
}

export const CompleteStep = ({ isSubmitting, onComplete, onBack }: CompleteStepProps) => {
  return (
    <div className="space-y-4">
      <div className="flex flex-col items-center gap-4 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-green-500/10">
          <PartyPopper className="size-8 text-green-500" />
        </div>
        <div className="space-y-1">
          <h2 className="text-lg font-semibold">You&apos;re All Set!</h2>
          <p className="text-sm text-muted-foreground">
            Your workspace is configured and ready to go. Click Finish to create
            everything and head to the dashboard.
          </p>
        </div>
      </div>

      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onBack} className="flex-1">
          Back
        </Button>
        <Button onClick={onComplete} disabled={isSubmitting} className="flex-1">
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 size-4 animate-spin" />
              Setting up...
            </>
          ) : (
            "Finish"
          )}
        </Button>
      </div>
    </div>
  )
}
