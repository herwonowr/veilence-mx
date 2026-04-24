"use client"

import { Telescope, CheckCircle2 } from "lucide-react"
import { Button } from "@/ui"
import type { DiscoveryPreference } from "@/features/onboarding/hooks/use-onboarding"

interface FirstDiscoveryStepProps {
  discoveryPreference: DiscoveryPreference
  onTriggerDiscovery: (ecosystem?: string) => void
  onNext: () => void
  onSkip: () => void
  onBack: () => void
}

export const FirstDiscoveryStep = ({
  discoveryPreference,
  onTriggerDiscovery,
  onNext,
  onSkip,
  onBack,
}: FirstDiscoveryStepProps) => {
  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">First Discovery</h2>
        <p className="text-sm text-muted-foreground">
          Trigger a discovery scan to fetch the most popular packages from PyPI and NPM
          registries based on your scan depth setting.
        </p>
      </div>

      <div className="flex flex-col items-center gap-4 rounded-lg border bg-muted/30 p-6">
        {discoveryPreference.enabled ? (
          <>
            <CheckCircle2 className="size-10 text-green-500" />
            <div className="text-center">
              <p className="font-medium">Discovery Enabled</p>
              <p className="text-sm text-muted-foreground">
                Discovery will run after setup completes. Packages will appear in
                your dashboard shortly after.
              </p>
            </div>
          </>
        ) : (
          <>
            <Telescope className="size-10 text-muted-foreground" />
            <div className="text-center">
              <p className="font-medium">Ready to Discover</p>
              <p className="text-sm text-muted-foreground">
                This will fetch top packages from both PyPI and NPM registries.
              </p>
            </div>
            <Button onClick={() => onTriggerDiscovery()}>
              Enable Discovery
            </Button>
          </>
        )}
      </div>

      {!discoveryPreference.enabled && (
        <Button type="button" variant="outline" onClick={onSkip} className="w-full">
          Skip
        </Button>
      )}

      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onBack} className="flex-1">
          Back
        </Button>
        <Button
          type="button"
          onClick={onNext}
          className="flex-1"
        >
          Continue
        </Button>
      </div>
    </div>
  )
}
