"use client"

import { useEffect, useState } from "react"
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
} from "@/components/ui/sidebar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { getDashboardStats } from "@/lib/api-client"
import { useAuth } from "@/lib/auth-context"
import { OrgSelector } from "@/components/org-selector"
import type { LucideIcon } from "lucide-react"

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
  const [alertCount, setAlertCount] = useState(0)
  const { user, isAuthenticated, logout } = useAuth()

  useEffect(() => {
    if (!isAuthenticated) return

    function fetchAlertCount() {
      getDashboardStats()
        .then((res) => {
          setAlertCount(res.data?.activeAlerts ?? 0)
        })
        .catch(() => {
          // Sidebar badge fetch — non-critical, no user action involved
        })
    }

    fetchAlertCount()
    const interval = setInterval(fetchAlertCount, 30_000)
    return () => clearInterval(interval)
  }, [isAuthenticated])

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
                  <User className="size-4" />
                  <div className="flex flex-col text-left text-sm leading-tight">
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
