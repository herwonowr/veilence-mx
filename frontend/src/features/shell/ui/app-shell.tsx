"use client"

import { usePathname } from "next/navigation"
import { SidebarInset, SidebarProvider } from "@/ui/components/sidebar"
import { useAuth } from "@/core/providers/auth-provider"
import { SiteHeader } from "@/features/shell/ui/site-header"
import { AppSidebar } from "@/features/shell/ui/app-sidebar"
import { CommandPalette } from "@/ui/layout/command-palette"

const AUTH_ROUTES = ["/login", "/register", "/forgot-password", "/reset-password"]

export const AppShell = ({
  children,
  notificationSlot,
  orgSelectorSlot,
}: {
  children: React.ReactNode
  notificationSlot?: React.ReactNode
  orgSelectorSlot?: React.ReactNode
}) => {
  const pathname = usePathname()
  const { isAuthenticated, isLoading } = useAuth()
  const isAuthRoute = AUTH_ROUTES.some((r) => pathname.startsWith(r))

  // For auth routes, always render plain (no app chrome)
  if (isAuthRoute) {
    return <main id="main-content">{children}</main>
  }

  // For protected routes, don't render app chrome until auth is confirmed
  if (isLoading || !isAuthenticated) {
    return <main id="main-content">{children}</main>
  }

  // Authenticated - render full app shell
  return (
    <SidebarProvider>
      <AppSidebar orgSelectorSlot={orgSelectorSlot} />
      <SidebarInset className="min-w-0 overflow-hidden">
        <SiteHeader actionSlot={notificationSlot} />
        <main id="main-content" className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
      </SidebarInset>
      <CommandPalette />
    </SidebarProvider>
  )
}
