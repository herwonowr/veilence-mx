"use client"

import { useEffect, useSyncExternalStore } from "react"
import { useRouter, usePathname } from "next/navigation"
import { useAuth, ROUTES } from "@/core"
import { Skeleton } from "@/ui"

const emptySubscribe = () => () => {}
const getClientSnapshot = () => true
const getServerSnapshot = () => false

export const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading, user } = useAuth()
  const router = useRouter()
  const pathname = usePathname()
  const hasMounted = useSyncExternalStore(emptySubscribe, getClientSnapshot, getServerSnapshot)

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push(`${ROUTES.LOGIN}?redirect=${encodeURIComponent(pathname)}`)
    }
  }, [isLoading, isAuthenticated, router, pathname])

  // Guard: redirect to /change-password when mustChangePassword is true
  useEffect(() => {
    if (!isLoading && isAuthenticated && user?.mustChangePassword && pathname !== ROUTES.CHANGE_PASSWORD) {
      router.push(ROUTES.CHANGE_PASSWORD)
    }
  }, [isLoading, isAuthenticated, user, pathname, router])

  if (!hasMounted || isLoading) {
    return (
      <div className="flex-1 space-y-4 p-4 md:p-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full" />
          ))}
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return null
  }

  // Block rendering while mustChangePassword and not on /change-password
  if (user?.mustChangePassword && pathname !== ROUTES.CHANGE_PASSWORD) {
    return null
  }

  return <>{children}</>
}
