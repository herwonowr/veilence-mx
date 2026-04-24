"use client"

import { PartyPopper } from "lucide-react"
import { Button } from "@/ui"

interface CompleteStepProps {
  onComplete: () => void
}

export const CompleteStep = ({ onComplete }: CompleteStepProps) => {
  return (
    <div className="flex flex-col items-center gap-6 py-4 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-green-500/10">
        <PartyPopper className="size-8 text-green-500" />
      </div>
      <div className="space-y-2">
        <h2 className="text-xl font-semibold">You&apos;re All Set!</h2>
        <p className="mx-auto max-w-sm text-sm text-muted-foreground">
          Your workspace is configured and ready to go. Head to the dashboard to see
          your monitored packages and any alerts.
        </p>
      </div>
      <Button onClick={onComplete} className="w-full">
        Go to Dashboard
      </Button>
    </div>
  )
}
