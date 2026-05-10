"use client"

import React from "react"
import { usePathname } from "next/navigation"
import Link from "next/link"
import { ROUTES } from "@/core"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/ui"
import { Separator, Button, SidebarTrigger, ThemeToggle, Kbd } from "@/ui"

const pageTitles: Record<string, string> = {
  [ROUTES.DASHBOARD]: "Dashboard",
  [ROUTES.PACKAGES]: "Packages",
  [ROUTES.ALERTS]: "Alerts",
  [ROUTES.RELEASES]: "Releases",
  [ROUTES.SETTINGS]: "Settings",
  [ROUTES.WORKSPACES]: "Workspaces",
  [ROUTES.ACCOUNT]: "Account",
  [ROUTES.ADMIN]: "Admin",
  [ROUTES.ADMIN_USERS]: "Users",
  [ROUTES.ADMIN_AUDIT_LOGS]: "Audit Logs",
  [ROUTES.ADMIN_SECURITY]: "Security",
  [ROUTES.SETTINGS_API_KEYS]: "API Keys",
  [ROUTES.SETTINGS_SESSIONS]: "Sessions",
  [ROUTES.SETTINGS_NOTIFICATIONS]: "Channels",
  [ROUTES.SETTINGS_QUEUE]: "Queue Monitor",
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
        // For ID segments like /workspaces/123, label as "Detail"
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
        <Button
          variant="outline"
          size="sm"
          type="button"
          onClick={() => {
            document.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true, bubbles: true }))
          }}
          className="hidden items-center gap-1 rounded-md border bg-muted/50 px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:inline-flex"
          aria-label="Open command palette"
        >
          <Kbd>⌘</Kbd>
          <Kbd>K</Kbd>
        </Button>
        {actionSlot}
        <ThemeToggle />
      </div>
    </header>
  )
}
