"use client"

import React, { useCallback, useEffect, useRef, useState } from "react"
import { useRouter } from "next/navigation"
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog"
import {
  LayoutDashboard,
  Package,
  ShieldAlert,
  BellDot,
  Radio,
  Settings,
  Activity,
  Layers,
  Key,
  User,
  Users,
  Workflow,
  Monitor,
  Search,
  Shield,
  ScrollText,
} from "lucide-react"
import { cn, useAuth, useCurrentWorkspaceRole, hasMinimumRole, ROUTES } from "@/core"
import { Kbd } from "@/ui"
import type { LucideIcon } from "lucide-react"

interface CommandItem {
  id: string
  label: string
  href: string
  icon: LucideIcon
  group: string
  keywords?: string[]
  /** Minimum workspace role required to see this item. Defaults to visible for all. */
  minRole?: "viewer" | "member" | "admin" | "owner"
  /** If true, only visible to platform super-admins (overrides minRole). */
  superAdminOnly?: boolean
}

const commandItems: CommandItem[] = [
  // Navigation
  { id: "dashboard", label: "Dashboard", href: ROUTES.DASHBOARD, icon: LayoutDashboard, group: "Navigation", keywords: ["home", "overview", "stats"] },
  { id: "packages", label: "Packages", href: ROUTES.PACKAGES, icon: Package, group: "Navigation", keywords: ["python", "npm", "go", "golang", "dependencies"] },
  { id: "releases", label: "Releases", href: ROUTES.RELEASES, icon: Activity, group: "Navigation", keywords: ["versions", "updates"] },
  { id: "alerts", label: "Alerts", href: ROUTES.ALERTS, icon: ShieldAlert, group: "Navigation", keywords: ["notifications", "warnings", "threats"] },
  { id: "notifications", label: "Notifications", href: ROUTES.NOTIFICATIONS, icon: BellDot, group: "Navigation", keywords: ["inbox", "messages", "updates"] },
  { id: "workspaces", label: "Workspaces", href: ROUTES.WORKSPACES, icon: Layers, group: "Navigation", keywords: ["teams"] },
  // Management
  { id: "settings", label: "Settings", href: ROUTES.SETTINGS, icon: Settings, group: "Management", keywords: ["preferences", "configuration"], minRole: "admin" },
  { id: "channels", label: "Channels", href: ROUTES.SETTINGS_NOTIFICATIONS, icon: Radio, group: "Management", keywords: ["notifications", "webhooks", "slack"], minRole: "admin" },
  { id: "queue", label: "Queue Monitor", href: ROUTES.SETTINGS_QUEUE, icon: Workflow, group: "Management", keywords: ["jobs", "workers", "processing"], minRole: "admin" },
  { id: "api-keys", label: "API Keys", href: ROUTES.SETTINGS_API_KEYS, icon: Key, group: "Management", keywords: ["tokens", "authentication"] },
  // Admin (super-admin only)
  { id: "security", label: "Security", href: ROUTES.ADMIN_SECURITY, icon: Shield, group: "Admin", keywords: ["sso", "saml", "authentication", "login"], superAdminOnly: true },
  { id: "users", label: "Users", href: ROUTES.ADMIN_USERS, icon: Users, group: "Admin", keywords: ["platform", "admin", "accounts"], superAdminOnly: true },
  { id: "audit-logs", label: "Audit Logs", href: ROUTES.ADMIN_AUDIT_LOGS, icon: ScrollText, group: "Admin", keywords: ["audit", "logs", "activity", "history"], superAdminOnly: true },
  // Account
  { id: "account", label: "Account", href: ROUTES.ACCOUNT, icon: User, group: "Account", keywords: ["profile", "email", "password"] },
  { id: "sessions", label: "Sessions", href: ROUTES.SETTINGS_SESSIONS, icon: Monitor, group: "Account", keywords: ["active", "devices"] },
]

export const CommandPalette = () => {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const router = useRouter()
  const { isAuthenticated, user } = useAuth()
  const { role } = useCurrentWorkspaceRole()

  // Filter items based on role permissions and super-admin status
  const permittedItems = React.useMemo(
    () => commandItems.filter((item) => {
      if (item.superAdminOnly) return !!user?.isSuperAdmin
      return !item.minRole || hasMinimumRole(role, item.minRole)
    }),
    [role, user?.isSuperAdmin]
  )

  // Filter items based on query
  const filteredItems = React.useMemo(() => {
    if (!query.trim()) return permittedItems
    const lowerQuery = query.toLowerCase()
    return permittedItems.filter(
      (item) =>
        item.label.toLowerCase().includes(lowerQuery) ||
        item.group.toLowerCase().includes(lowerQuery) ||
        item.keywords?.some((kw) => kw.includes(lowerQuery))
    )
  }, [query, permittedItems])

  // Group filtered items, preserving order and computing flat indices
  const groupedItems = React.useMemo(() => {
    const groups: { group: string; items: { item: CommandItem; flatIndex: number }[] }[] = []
    const groupMap = new Map<string, { item: CommandItem; flatIndex: number }[]>()
    let idx = 0
    for (const item of filteredItems) {
      let arr = groupMap.get(item.group)
      if (!arr) {
        arr = []
        groupMap.set(item.group, arr)
        groups.push({ group: item.group, items: arr })
      }
      arr.push({ item, flatIndex: idx++ })
    }
    return groups
  }, [filteredItems])

  // Reset state when dialog opens/closes
  const handleOpenChange = useCallback((isOpen: boolean) => {
    setOpen(isOpen)
    if (isOpen) {
      setQuery("")
      setSelectedIndex(0)
    }
  }, [])

  // Global Cmd+K / Ctrl+K listener
  useEffect(() => {
    if (!isAuthenticated) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault()
        e.stopPropagation()
        handleOpenChange(!open)
      }
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [isAuthenticated, open, handleOpenChange])

  // Navigate to the selected item
  const navigateTo = useCallback(
    (item: CommandItem) => {
      setOpen(false)
      router.push(item.href)
    },
    [router]
  )

  // Scroll the selected item into view
  useEffect(() => {
    if (!listRef.current) return
    const selectedEl = listRef.current.querySelector(
      `[data-command-index="${selectedIndex}"]`
    )
    if (selectedEl) {
      selectedEl.scrollIntoView({ block: "nearest" })
    }
  }, [selectedIndex])

  // Keyboard navigation within the palette
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown": {
          e.preventDefault()
          setSelectedIndex((prev) =>
            prev < filteredItems.length - 1 ? prev + 1 : 0
          )
          break
        }
        case "ArrowUp": {
          e.preventDefault()
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : filteredItems.length - 1
          )
          break
        }
        case "Enter": {
          e.preventDefault()
          const item = filteredItems[selectedIndex]
          if (item) navigateTo(item)
          break
        }
        case "Home": {
          e.preventDefault()
          setSelectedIndex(0)
          break
        }
        case "End": {
          e.preventDefault()
          setSelectedIndex(filteredItems.length - 1)
          break
        }
      }
    },
    [filteredItems, selectedIndex, navigateTo]
  )

  if (!isAuthenticated) return null

  return (
    <DialogPrimitive.Root open={open} onOpenChange={handleOpenChange} disablePointerDismissal>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Backdrop
          className="fixed inset-0 isolate z-50 bg-black/25 duration-100 supports-backdrop-filter:backdrop-blur-xs data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0"
        />
        <DialogPrimitive.Popup
          className="fixed inset-0 z-50 overflow-y-auto pt-[15vh] pb-8 outline-none"
          onKeyDown={handleKeyDown}
          onClick={(e) => {
            if (e.target === e.currentTarget) setOpen(false)
          }}
        >
          <div
            className="relative mx-auto w-full max-w-[calc(100%-2rem)] overflow-hidden rounded-xl bg-popover text-popover-foreground ring-1 ring-foreground/10 shadow-xl duration-100 sm:max-w-lg data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95"
          >
          <DialogPrimitive.Title className="sr-only">
              Command palette
            </DialogPrimitive.Title>
            <DialogPrimitive.Description className="sr-only">
              Search and navigate to pages. Use arrow keys to navigate, Enter to select, Escape to close.
            </DialogPrimitive.Description>

            {/* Search input */}
            <div className="flex items-center gap-2 border-b px-3">
              <Search className="size-4 shrink-0 text-muted-foreground" />
              <input
                ref={inputRef}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value)
                  setSelectedIndex(0)
                }}
                placeholder="Search pages..."
                autoFocus
                className="flex-1 bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground"
                aria-label="Search command palette"
                role="combobox"
                aria-expanded={true}
                aria-controls="command-palette-list"
                aria-activedescendant={
                  filteredItems[selectedIndex]
                    ? `command-item-${filteredItems[selectedIndex].id}`
                    : undefined
                }
              />
              <Kbd className="hidden sm:inline-block">
                Esc
              </Kbd>
            </div>

            {/* Results list */}
            <div
              ref={listRef}
              id="command-palette-list"
              role="listbox"
              className="max-h-72 overflow-y-auto p-1"
            >
              {filteredItems.length === 0 ? (
                <div className="py-6 text-center text-sm text-muted-foreground">
                  No results found
                </div>
              ) : (
                groupedItems.map(({ group, items }) => (
                  <div key={group} role="group" aria-label={group}>
                    <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                      {group}
                    </div>
                    {items.map(({ item, flatIndex: currentIndex }) => {
                      const isSelected = currentIndex === selectedIndex
                      return (
                        <div
                          key={item.id}
                          id={`command-item-${item.id}`}
                          role="option"
                          aria-selected={isSelected}
                          data-command-index={currentIndex}
                          className={cn(
                            "flex cursor-pointer items-center gap-2 rounded-lg px-2 py-2 text-sm outline-none select-none",
                            isSelected
                              ? "bg-accent text-accent-foreground"
                              : "text-foreground hover:bg-accent/50"
                          )}
                          onClick={() => navigateTo(item)}
                          onMouseEnter={() => setSelectedIndex(currentIndex)}
                        >
                          <item.icon className="size-4 shrink-0 text-muted-foreground" />
                          <span>{item.label}</span>
                        </div>
                      )
                    })}
                  </div>
                ))
              )}
            </div>

            {/* Footer hint */}
            <div className="flex items-center gap-3 border-t px-3 py-2">
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Kbd>↑</Kbd>
                <Kbd>↓</Kbd>
                <span>navigate</span>
              </span>
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Kbd>↵</Kbd>
                <span>select</span>
              </span>
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Kbd>esc</Kbd>
                <span>close</span>
              </span>
            </div>
          </div>
        </DialogPrimitive.Popup>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
