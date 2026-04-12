"use client"

import React, { useEffect, useRef } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Package,
  Bell,
  Settings,
  Shield,
  Activity,
  Building2,
  Key,
  LogOut,
  User,
  ListOrdered,
  Monitor,
} from "lucide-react"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarRail,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useAuth } from "@/lib/auth-context"
import { OrgSelector } from "@/components/org-selector"
import { useDashboardStats } from "@/features/dashboard"
import type { LucideIcon } from "lucide-react"

function UserAvatar({ name }: { name: string }) {
  const initial = name.charAt(0).toUpperCase()
  return (
    <span
      className="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-[11px] font-semibold text-primary-foreground"
      aria-hidden="true"
    >
      {initial}
    </span>
  )
}

interface NavItem {
  title: string
  href: string
  icon: LucideIcon
}

const navItems: NavItem[] = [
  { title: "Dashboard", href: "/", icon: LayoutDashboard },
  { title: "Packages", href: "/packages", icon: Package },
  { title: "Releases", href: "/releases", icon: Activity },
  { title: "Alerts", href: "/alerts", icon: Bell },
  { title: "Organizations", href: "/organizations", icon: Building2 },
]

const settingsItems: NavItem[] = [
  { title: "Settings", href: "/settings", icon: Settings },
  { title: "Channels", href: "/settings/notifications", icon: Bell },
  { title: "Queue", href: "/settings/queue", icon: ListOrdered },
  { title: "API Keys", href: "/settings/api-keys", icon: Key },
  { title: "Sessions", href: "/settings/sessions", icon: Monitor },
]

export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const pathname = usePathname()
  const { user, isAuthenticated, logout } = useAuth()
  const { setOpenMobile, isMobile } = useSidebar()

  // Close mobile sheet on route change
  const prevPathname = useRef(pathname)
  useEffect(() => {
    if (isMobile && prevPathname.current !== pathname) {
      setOpenMobile(false)
    }
    prevPathname.current = pathname
  }, [pathname, isMobile, setOpenMobile])

  const { data: statsRes } = useDashboardStats({
    refetchInterval: 30_000,
    enabled: isAuthenticated,
  })
  const alertCount = statsRes?.data?.activeAlerts ?? 0

  return (
    <Sidebar variant="inset" collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              render={<Link href="/" />}
              tooltip="Veilence-MX"
            >
              <Shield />
              <div className="flex flex-col">
                <span className="font-semibold">Veilence-MX</span>
                <span className="text-xs text-muted-foreground">
                  Supply Chain Monitor
                </span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        {isAuthenticated && (
          <>
            <SidebarSeparator />
            <OrgSelector />
          </>
        )}
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => {
                const isActive =
                  item.href === "/"
                    ? pathname === "/"
                    : pathname.startsWith(item.href)

                return (
                  <SidebarMenuItem key={item.href}>
                    <SidebarMenuButton
                      render={<Link href={item.href} />}
                      isActive={isActive}
                      tooltip={item.title}
                    >
                      <item.icon />
                      <span>{item.title}</span>
                    </SidebarMenuButton>
                    {item.href === "/alerts" && alertCount > 0 && (
                      <SidebarMenuBadge>{alertCount}</SidebarMenuBadge>
                    )}
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Management</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {settingsItems.map((item) => {
                const isActive =
                  item.href === "/settings"
                    ? pathname === "/settings"
                    : pathname === item.href || pathname.startsWith(item.href + "/")
                return (
                  <SidebarMenuItem key={item.href}>
                    <SidebarMenuButton
                      render={<Link href={item.href} />}
                      isActive={isActive}
                      tooltip={item.title}
                    >
                      <item.icon />
                      <span>{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        {isAuthenticated && user ? (
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <SidebarMenuButton
                      tooltip={user.email}
                      className="w-full"
                    />
                  }
                >
                  <UserAvatar name={user.firstName || user.email} />
                  <div className="flex min-w-0 flex-1 flex-col text-left text-sm leading-tight">
                    <span className="truncate font-medium">
                      {user.firstName} {user.lastName}
                    </span>
                    <span className="truncate text-xs text-muted-foreground">
                      {user.email}
                    </span>
                  </div>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="start"
                  side="top"
                  sideOffset={4}
                  className="w-56"
                >
                  <DropdownMenuItem
                    render={<Link href="/account" />}
                  >
                    <User className="mr-2 size-4" />
                    Account
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => {
                      logout()
                    }}
                  >
                    <LogOut className="mr-2 size-4" />
                    Sign out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        ) : (
          <span className="px-2 text-xs text-muted-foreground">v0.1.0</span>
        )}
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  )
}
