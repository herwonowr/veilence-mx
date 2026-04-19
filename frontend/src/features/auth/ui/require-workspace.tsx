"use client"

import { useRouter } from "next/navigation"
import { useAuth } from "@/core"
import { Building2 } from "lucide-react"
import { Button } from "@/ui/components/button"

interface RequireWorkspaceProps {
  children: React.ReactNode
  feature?: string
}

export const RequireWorkspace = ({ children, feature }: RequireWorkspaceProps) => {
  const { currentWorkspace, workspaces } = useAuth()
  const router = useRouter()

  if (currentWorkspace) {
    return <>{children}</>
  }

  const hasAnyWorkspace = workspaces.length > 0

  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Building2 className="h-10 w-10 text-muted-foreground mb-4" />
      <h2 className="text-xl font-semibold mb-2">
        {hasAnyWorkspace ? "No workspace selected" : "Workspace required"}
      </h2>
      <p className="text-muted-foreground mb-6 max-w-sm">
        {hasAnyWorkspace
          ? `Select a workspace from the sidebar to view ${feature ?? "this page"}.`
          : `Create a workspace to start viewing ${feature ?? "this page"}.`}
      </p>
      {!hasAnyWorkspace && (
        <Button onClick={() => router.push("/workspaces")}>
          Create Workspace
        </Button>
      )}
    </div>
  )
}
