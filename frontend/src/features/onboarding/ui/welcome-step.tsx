"use client"

import { ShieldCheck, LogOut } from "lucide-react"
import { Button } from "@/ui"
import { useAuth } from "@/core"

interface WelcomeStepProps {
  onNext: () => void
}

export const WelcomeStep = ({ onNext }: WelcomeStepProps) => {
  const { logout } = useAuth()

  return (
    <div className="space-y-4">
      <div className="flex flex-col items-center gap-4 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10">
          <ShieldCheck className="size-8 text-primary" />
        </div>
        <div className="space-y-1">
          <h2 className="text-lg font-semibold">Welcome to Veilence-MX</h2>
          <p className="text-sm text-muted-foreground">
            Real-time supply chain monitoring for PyPI and NPM. We&apos;ll help you set up
            your workspace and start monitoring packages in just a few steps.
          </p>
        </div>
      </div>

      <div className="space-y-1 text-left text-sm text-muted-foreground">
        <p>Here&apos;s what we&apos;ll set up:</p>
        <ul className="ml-4 list-disc space-y-1">
          <li>Create your workspace</li>
          <li>Configure monitoring settings</li>
          <li>Add packages to monitor</li>
          <li>Run your first discovery scan</li>
        </ul>
      </div>

      <Button onClick={onNext} className="w-full">
        Get Started
      </Button>
      <Button variant="outline" onClick={logout} className="w-full">
        <LogOut className="mr-1 h-4 w-4" />
        Log out
      </Button>
    </div>
  )
}
