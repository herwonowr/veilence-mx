"use client"

import { useOnboardingCheck } from "@/features/onboarding/hooks/use-onboarding-check"
import { OnboardingWizard } from "@/features/onboarding/ui/onboarding-wizard"

export const OnboardingGate = ({ children }: { children: React.ReactNode }) => {
  const { shouldShowOnboarding } = useOnboardingCheck()

  return (
    <>
      {children}
      {shouldShowOnboarding && <OnboardingWizard />}
    </>
  )
}
