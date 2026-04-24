"use client"

import { cn } from "@/core"
import { CheckIcon } from "lucide-react"

interface WizardProgressProps {
  currentStep: number
  totalSteps: number
  labels?: string[]
}

const defaultLabels = [
  "Welcome",
  "Workspace",
  "Settings",
  "Packages",
  "Discovery",
  "Complete",
]

export const WizardProgress = ({
  currentStep,
  totalSteps,
  labels = defaultLabels,
}: WizardProgressProps) => {
  return (
    <div className="flex items-center justify-center gap-2 px-4">
      {Array.from({ length: totalSteps }, (_, i) => {
        const step = i + 1
        const isCompleted = step < currentStep
        const isCurrent = step === currentStep

        return (
          <div key={step} className="flex items-center gap-2">
            <div className="flex flex-col items-center gap-1">
              <div
                className={cn(
                  "flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium transition-colors",
                  isCompleted && "bg-primary text-primary-foreground",
                  isCurrent && "bg-primary text-primary-foreground ring-2 ring-primary/30",
                  !isCompleted && !isCurrent && "bg-muted text-muted-foreground"
                )}
              >
                {isCompleted ? <CheckIcon className="size-3.5" /> : step}
              </div>
              <span
                className={cn(
                  "text-[10px] leading-tight",
                  isCurrent ? "text-foreground font-medium" : "text-muted-foreground"
                )}
              >
                {labels[i]}
              </span>
            </div>
            {i < totalSteps - 1 && (
              <div
                className={cn(
                  "mb-4 h-px w-6 transition-colors",
                  step < currentStep ? "bg-primary" : "bg-border"
                )}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}
