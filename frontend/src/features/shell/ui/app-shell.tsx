"use client"

import { usePathname } from "next/navigation"
import { SidebarInset, SidebarProvider } from "@/ui"
import { CommandPalette } from "@/features/shell/ui/command-palette"
import { useAuth, NO_CHROME_PATHS } from "@/core"
import { SiteHeader } from "@/features/shell/ui/site-header"
import { AppSidebar } from "@/features/shell/ui/app-sidebar"

export const AppShell = ({
  children,
  notificationSlot,
  workspaceSelectorSlot,
  sidebarDefaultOpen = true,
}: {
  children: React.ReactNode
  notificationSlot?: React.ReactNode
  workspaceSelectorSlot?: React.ReactNode
  sidebarDefaultOpen?: boolean
}) => {
  const pathname = usePathname()
  const { isAuthenticated, isLoading } = useAuth()
  const isAuthRoute = NO_CHROME_PATHS.some((r) => pathname.startsWith(r))

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
    <SidebarProvider defaultOpen={sidebarDefaultOpen}>
      <AppSidebar workspaceSelectorSlot={workspaceSelectorSlot} />
      <SidebarInset className="min-w-0 overflow-hidden">
        <SiteHeader actionSlot={notificationSlot} />
        <main id="main-content" className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
      </SidebarInset>
      <CommandPalette />
    </SidebarProvider>
  )
}
