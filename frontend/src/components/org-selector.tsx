"use client"

import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { Building2, ChevronsUpDown, Plus, AlertCircle, Loader2 } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

export function OrgSelector() {
  const { currentOrg, organizations, orgsLoading, orgsError, setCurrentOrg, refreshOrgs } = useAuth()
  const router = useRouter()

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                className="w-full data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                tooltip="Organization"
                aria-label={`Current organization: ${currentOrg?.name ?? "None selected"}`}
              />
            }
          >
            {orgsLoading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Building2 className="size-4" />
            )}
            <div className="flex flex-1 flex-col text-left text-sm leading-tight">
              <span className="truncate font-medium">
                {orgsLoading
                  ? "Loading..."
                  : currentOrg?.name ?? "No Organization"}
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
            <DropdownMenuGroup>
              <DropdownMenuLabel>Organizations</DropdownMenuLabel>
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
              {orgsLoading && (
                <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-muted-foreground">
                  <Loader2 className="size-3 animate-spin" />
                  Loading organizations...
                </div>
              )}
              {!orgsLoading && !orgsError && organizations.map((org) => (
                <DropdownMenuItem
                  key={org.id}
                  onSelect={() => {
                    setCurrentOrg(org)
                    router.refresh()
                  }}
                >
                  <Building2 className="mr-2 size-4" />
                  <span className="truncate">{org.name}</span>
                </DropdownMenuItem>
              ))}
              {!orgsLoading && !orgsError && organizations.length === 0 && (
                <div className="px-2 py-1.5 text-xs text-muted-foreground">
                  No organizations yet
                </div>
              )}
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={() => router.push("/organizations?create=true")}
            >
              <Plus className="mr-2 size-4" />
              Create Organization
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
