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
} from "@/ui/components/breadcrumb"
import { Separator } from "@/ui/components/separator"
import { SidebarTrigger } from "@/ui/components/sidebar"
import { ThemeToggle } from "@/ui/layout/theme-toggle"

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/packages": "Packages",
  "/alerts": "Alerts",
  "/releases": "Releases",
  "/settings": "Settings",
  "/organizations": "Organizations",
  "/account": "Account",
  "/settings/api-keys": "API Keys",
  "/settings/sessions": "Sessions",
  "/settings/notifications": "Channels",
  "/settings/queue": "Queue Monitor",
}

/** Capitalize and humanize a URL segment (e.g. "api-keys" -> "API Keys") */
const humanizeSegment = (segment: string): string =>
  decodeURIComponent(segment)
    .replace(/-/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase())

/** Detect if a segment looks like an ID (numeric or UUID-like) */
const isIdSegment = (segment: string): boolean =>
  /^\d+$/.test(segment) || /^[0-9a-f-]{8,}$/i.test(segment)

export const SiteHeader = ({ actionSlot }: { actionSlot?: React.ReactNode }) => {
  const pathname = usePathname()

  const segments = pathname.split("/").filter(Boolean)
  const crumbs: { label: string; href: string }[] = []

  if (segments.length === 0) {
    crumbs.push({ label: "Dashboard", href: "/" })
  } else {
    for (let i = 0; i < segments.length; i++) {
      const href = "/" + segments.slice(0, i + 1).join("/")
      const segment = segments[i]

      // Use known title map first
      if (pageTitles[href]) {
        crumbs.push({ label: pageTitles[href], href })
      } else if (isIdSegment(segment)) {
        // For ID segments like /organizations/123, label as "Detail"
        const parentLabel = crumbs.length > 0 ? crumbs[crumbs.length - 1].label : ""
        crumbs.push({ label: `${parentLabel} Detail`.trim(), href })
      } else if (segment === "audit") {
        crumbs.push({ label: "Audit Log", href })
      } else {
        crumbs.push({ label: humanizeSegment(segment), href })
      }
    }
  }

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" aria-label="Toggle sidebar" />
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
        {actionSlot}
        <ThemeToggle />
      </div>
    </header>
  )
}
