"use client"

import React, { useCallback, useEffect, useRef, useState } from "react"
import { useRouter } from "next/navigation"
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog"
import {
  LayoutDashboard,
  Package,
  Bell,
  Settings,
  Activity,
  Building2,
  Key,
  User,
  ListOrdered,
  Monitor,
  Search,
} from "lucide-react"
import { cn } from "@/core/utils"
import { useAuth } from "@/core/providers/auth-provider"
import type { LucideIcon } from "lucide-react"

interface CommandItem {
  id: string
  label: string
  href: string
  icon: LucideIcon
  group: string
  keywords?: string[]
}

const commandItems: CommandItem[] = [
  // Navigation
  { id: "dashboard", label: "Dashboard", href: "/", icon: LayoutDashboard, group: "Navigation", keywords: ["home", "overview", "stats"] },
  { id: "packages", label: "Packages", href: "/packages", icon: Package, group: "Navigation", keywords: ["python", "npm", "dependencies"] },
  { id: "releases", label: "Releases", href: "/releases", icon: Activity, group: "Navigation", keywords: ["versions", "updates"] },
  { id: "alerts", label: "Alerts", href: "/alerts", icon: Bell, group: "Navigation", keywords: ["notifications", "warnings", "threats"] },
  { id: "organizations", label: "Organizations", href: "/organizations", icon: Building2, group: "Navigation", keywords: ["orgs", "teams"] },
  // Management
  { id: "settings", label: "Settings", href: "/settings", icon: Settings, group: "Management", keywords: ["preferences", "configuration"] },
  { id: "channels", label: "Channels", href: "/settings/notifications", icon: Bell, group: "Management", keywords: ["notifications", "webhooks", "slack"] },
  { id: "queue", label: "Queue Monitor", href: "/settings/queue", icon: ListOrdered, group: "Management", keywords: ["jobs", "workers", "processing"] },
  { id: "api-keys", label: "API Keys", href: "/settings/api-keys", icon: Key, group: "Management", keywords: ["tokens", "authentication"] },
  { id: "sessions", label: "Sessions", href: "/settings/sessions", icon: Monitor, group: "Management", keywords: ["active", "devices"] },
  // Account
  { id: "account", label: "Account", href: "/account", icon: User, group: "Account", keywords: ["profile", "email", "password"] },
]

export const CommandPalette = () => {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const router = useRouter()
  const { isAuthenticated } = useAuth()

  // Filter items based on query
  const filteredItems = React.useMemo(() => {
    if (!query.trim()) return commandItems
    const lowerQuery = query.toLowerCase()
    return commandItems.filter(
      (item) =>
        item.label.toLowerCase().includes(lowerQuery) ||
        item.group.toLowerCase().includes(lowerQuery) ||
        item.keywords?.some((kw) => kw.includes(lowerQuery))
    )
  }, [query])

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
    <DialogPrimitive.Root open={open} onOpenChange={handleOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Backdrop
          className="fixed inset-0 isolate z-50 bg-black/25 duration-100 supports-backdrop-filter:backdrop-blur-xs data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0"
        />
        <DialogPrimitive.Popup
          className="fixed top-[20%] left-1/2 z-50 w-full max-w-[calc(100%-2rem)] -translate-x-1/2 overflow-hidden rounded-xl bg-popover text-popover-foreground ring-1 ring-foreground/10 shadow-xl duration-100 outline-none sm:max-w-lg data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95"
          onKeyDown={handleKeyDown}
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
            <kbd className="pointer-events-none hidden select-none rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] font-medium text-muted-foreground sm:inline-block">
              Esc
            </kbd>
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
              <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">↑</kbd>
              <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">↓</kbd>
              <span>navigate</span>
            </span>
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">↵</kbd>
              <span>select</span>
            </span>
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <kbd className="rounded border bg-muted px-1 py-0.5 font-mono text-[10px]">esc</kbd>
              <span>close</span>
            </span>
          </div>
        </DialogPrimitive.Popup>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
