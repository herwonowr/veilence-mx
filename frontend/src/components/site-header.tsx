"use client"

import React from "react"
import { usePathname } from "next/navigation"
import Link from "next/link"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { ThemeToggle } from "@/components/theme-toggle"
import { NotificationBell } from "@/components/notification-bell"

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/packages": "Packages",
  "/alerts": "Alerts",
  "/settings": "Settings",
  "/organizations": "Organizations",
  "/settings/api-keys": "API Keys",
}

export function SiteHeader() {
  const pathname = usePathname()

  const segments = pathname.split("/").filter(Boolean)
  const crumbs: { label: string; href: string }[] = []

  if (segments.length === 0) {
    crumbs.push({ label: "Dashboard", href: "/" })
  } else {
    for (let i = 0; i < segments.length; i++) {
      const href = "/" + segments.slice(0, i + 1).join("/")
      const label = pageTitles[href] ?? decodeURIComponent(segments[i])
      crumbs.push({ label, href })
    }
  }

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="mr-2 data-vertical:self-center h-4!" />
      <Breadcrumb>
        <BreadcrumbList>
          {crumbs.map((crumb, index) => {
            const isLast = index === crumbs.length - 1
            return (
              <React.Fragment key={crumb.href}>
                {index > 0 && <BreadcrumbSeparator />}
                <BreadcrumbItem>
                  {isLast ? (
                    <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink render={<Link href={crumb.href} />}>
                      {crumb.label}
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
              </React.Fragment>
            )
          })}
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto flex items-center gap-1">
        <NotificationBell />
        <ThemeToggle />
      </div>
    </header>
  )
}
