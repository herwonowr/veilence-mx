"use client"

import { Button } from "@/ui"
import { Zap, Loader2 } from "lucide-react"
import { useTestPlatformSSOConfig } from "@/features/sso-admin/hooks/use-sso-configs"

interface SSOTestButtonProps {
  configId: string
}

export const SSOTestButton = ({ configId }: SSOTestButtonProps) => {
  const testMutation = useTestPlatformSSOConfig()

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      disabled={testMutation.isPending}
      onClick={() => testMutation.mutate(configId)}
      aria-label="Test configuration"
      title="Test configuration"
    >
      {testMutation.isPending ? (
        <Loader2 className="size-4 animate-spin" />
      ) : (
        <Zap className="size-4 text-primary" />
      )}
    </Button>
  )
}
