"use client"

import { useState, useMemo } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/core/providers/auth-provider"
import { Building2, ChevronsUpDown, Plus, Settings, AlertCircle, Loader2, Check } from "lucide-react"
import { Input } from "@/ui/components/input"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/components/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/ui/components/sidebar"

const SEARCH_THRESHOLD = 5

export const OrgSelector = () => {
  const { currentOrg, organizations, orgsLoading, orgsError, setCurrentOrg, refreshOrgs } = useAuth()
  const router = useRouter()
  const [search, setSearch] = useState("")

  const hasOrgs = organizations.length > 0
  const showSearch = organizations.length > SEARCH_THRESHOLD

  const filteredOrgs = useMemo(() => {
    if (!search) return organizations
    const q = search.toLowerCase()
    return organizations.filter(
      (org) => org.name.toLowerCase().includes(q) || org.slug.toLowerCase().includes(q)
    )
  }, [organizations, search])

  // Reset search when dropdown closes
  const handleOpenChange = (open: boolean) => {
    if (!open) setSearch("")
  }

  // ─── No orgs: pulsing CTA trigger ───
  const noOrgsIdle = !orgsLoading && !orgsError && !hasOrgs

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu onOpenChange={handleOpenChange}>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                className={`w-full data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground ${
                  noOrgsIdle ? "animate-pulse border border-dashed border-green-500/60" : ""
                }`}
                tooltip="Organization"
                aria-label={`Current organization: ${currentOrg?.name ?? "None selected"}`}
              />
            }
          >
            {orgsLoading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : noOrgsIdle ? (
              <Plus className="size-4 text-green-500" />
            ) : (
              <Building2 className="size-4" />
            )}
            <div className="flex flex-1 flex-col text-left text-sm leading-tight">
              <span className="truncate font-medium">
                {orgsLoading
                  ? "Loading..."
                  : noOrgsIdle
                    ? "Create Organization"
                    : currentOrg?.name ?? "Select Organization"}
              </span>
              {currentOrg?.slug && !orgsLoading && (
                <span className="truncate text-xs text-muted-foreground">
                  {currentOrg.slug}
                </span>
              )}
            </div>
            <ChevronsUpDown className="ml-auto size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-56"
            align="start"
            side="bottom"
            sideOffset={4}
          >
            {/* Search input (only when >5 orgs) */}
            {showSearch && (
              <div className="px-1.5 pb-1.5 pt-0.5">
                <Input
                  placeholder="Search organizations..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="h-7 text-xs"
                  aria-label="Filter organizations"
                  autoFocus
                />
              </div>
            )}

            <DropdownMenuGroup>
              <DropdownMenuLabel>Organizations</DropdownMenuLabel>

              {/* Error state */}
              {orgsError && (
                <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-destructive">
                  <AlertCircle className="size-3 shrink-0" />
                  <span className="truncate">{orgsError}</span>
                  <button
                    type="button"
                    className="ml-auto shrink-0 text-xs underline hover:no-underline"
                    onClick={() => refreshOrgs()}
                  >
                    Retry
                  </button>
                </div>
              )}

              {/* Loading state */}
              {orgsLoading && (
                <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-muted-foreground">
                  <Loader2 className="size-3 animate-spin" />
                  Loading organizations...
                </div>
              )}

              {/* Org list with radio-style checkmarks */}
              {!orgsLoading && !orgsError && filteredOrgs.map((org) => (
                <DropdownMenuItem
                  key={org.id}
                  onClick={() => {
                    setCurrentOrg(org)
                    router.refresh()
                  }}
                >
                  <Building2 className="mr-2 size-4 shrink-0" />
                  <span className="truncate">{org.name}</span>
                  {currentOrg?.id === org.id && (
                    <Check className="ml-auto size-4 shrink-0 text-muted-foreground" />
                  )}
                </DropdownMenuItem>
              ))}

              {/* Search yielded no results */}
              {!orgsLoading && !orgsError && hasOrgs && filteredOrgs.length === 0 && (
                <div className="px-2 py-1.5 text-xs text-muted-foreground">
                  No organizations match &ldquo;{search}&rdquo;
                </div>
              )}

              {/* Empty state (no orgs at all) */}
              {noOrgsIdle && (
                <div className="px-2 py-1.5 text-xs text-muted-foreground">
                  No organizations yet
                </div>
              )}
            </DropdownMenuGroup>

            <DropdownMenuSeparator />

            {/* Action links */}
            <DropdownMenuItem
              onClick={() => router.push("/organizations?create=true")}
            >
              <Plus className="mr-2 size-4" />
              Create Organization
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => router.push("/organizations")}
            >
              <Settings className="mr-2 size-4" />
              Manage Organizations
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
