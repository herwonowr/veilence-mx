"use client"

import { useCurrentWorkspaceRole, hasMinimumRole } from "@/core/hooks/use-workspace-role"
import type { WorkspaceRole } from "@/core/hooks/use-workspace-role"
import { Skeleton } from "@/ui/components/skeleton"
import { ShieldX } from "lucide-react"

/**
 * RequireRole — gates children behind a minimum workspace role.
 *
 * NOTE: Frontend role checks are UI convenience only. The backend enforces
 * authorization on every API endpoint. This component prevents viewers from
 * seeing admin-only pages, but does not replace server-side checks.
 */

interface RequireRoleProps {
  minimumRole: WorkspaceRole
  children: React.ReactNode
  /** Shown when the user lacks the required role. Defaults to an Access Denied page. */
  fallback?: React.ReactNode
}

const AccessDenied = () => (
  <div className="flex flex-1 flex-col items-center justify-center gap-4 py-24">
    <ShieldX className="size-12 text-muted-foreground" />
    <h2 className="text-xl font-semibold">Access Denied</h2>
    <p className="text-sm text-muted-foreground">
      You don&apos;t have permission to view this page.
    </p>
  </div>
)

const RoleSkeleton = () => (
  <div className="flex-1 space-y-4 p-4 md:p-6">
    <Skeleton className="h-8 w-48" />
    <Skeleton className="h-64 w-full" />
  </div>
)

export const RequireRole = ({
  minimumRole,
  children,
  fallback,
}: RequireRoleProps) => {
  const { role, isLoading } = useCurrentWorkspaceRole()

  if (isLoading) {
    return <RoleSkeleton />
  }

  if (!hasMinimumRole(role, minimumRole)) {
    return <>{fallback ?? <AccessDenied />}</>
  }

  return <>{children}</>
}
