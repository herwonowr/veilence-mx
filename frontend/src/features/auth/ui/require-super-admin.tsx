"use client"

import { useAuth } from "@/core"
import { Skeleton } from "@/ui"
import { ShieldX } from "lucide-react"

/**
 * RequireSuperAdmin - gates children behind the isSuperAdmin flag on the current user.
 *
 * NOTE: Frontend checks are UI convenience only. The backend enforces
 * authorization on every API endpoint.
 */

interface RequireSuperAdminProps {
  children: React.ReactNode
  /** Shown when the user is not a super admin. Defaults to an Access Denied page. */
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

const SuperAdminSkeleton = () => (
  <div className="flex-1 space-y-4 p-4 md:p-6">
    <Skeleton className="h-8 w-48" />
    <Skeleton className="h-64 w-full" />
  </div>
)

export const RequireSuperAdmin = ({
  children,
  fallback,
}: RequireSuperAdminProps) => {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return <SuperAdminSkeleton />
  }

  if (!user?.isSuperAdmin) {
    return <>{fallback ?? <AccessDenied />}</>
  }

  return <>{children}</>
}
