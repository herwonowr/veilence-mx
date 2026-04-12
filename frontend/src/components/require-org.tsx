"use client"

import { useAuth } from "@/lib/auth-context"
import { Building2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useRouter } from "next/navigation"

interface RequireOrgProps {
  children: React.ReactNode
  /** Page name shown in the message, e.g. "packages", "releases", "alerts" */
  feature?: string
}

/**
 * RequireOrg — guards org-scoped pages.
 *
 * When the user has no organization selected, shows a friendly prompt
 * instead of letting API calls fail with 400 "valid organization ID is required".
 */
export function RequireOrg({ children, feature }: RequireOrgProps) {
  const { currentOrg, organizations } = useAuth()
  const router = useRouter()

  if (currentOrg) {
    return <>{children}</>
  }

  const hasAnyOrg = organizations.length > 0

  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Building2 className="h-10 w-10 text-muted-foreground mb-4" />
      <h2 className="text-xl font-semibold mb-2">
        {hasAnyOrg ? "No organization selected" : "Organization required"}
      </h2>
      <p className="text-muted-foreground mb-6 max-w-sm">
        {hasAnyOrg
          ? `Select an organization from the sidebar to view ${feature ?? "this page"}.`
          : `Create an organization to start viewing ${feature ?? "this page"}.`}
      </p>
      {!hasAnyOrg && (
        <Button onClick={() => router.push("/organizations")}>
          Create Organization
        </Button>
      )}
    </div>
  )
}
