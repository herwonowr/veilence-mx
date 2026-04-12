"use client"

import { usePathname } from "next/navigation"
import { AppSidebar } from "@/components/app-sidebar"
import { CommandPalette } from "@/components/command-palette"
import { SiteHeader } from "@/components/site-header"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { useAuth } from "@/lib/auth-context"

const AUTH_ROUTES = ["/login", "/register", "/forgot-password", "/reset-password"]

export function AppShell({ children }: { children: React.ReactNode }) {
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

  // Authenticated — render full app shell
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="min-w-0 overflow-hidden">
        <SiteHeader />
        <main id="main-content" className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
      </SidebarInset>
      <CommandPalette />
    </SidebarProvider>
  )
}
