"use client"

import { useState, useMemo } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/core/providers/auth-provider"
import { Building2, ChevronsUpDown, Plus, Settings, AlertCircle, Loader2, Check } from "lucide-react"
import { Button } from "@/ui/components/button"
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

export const WorkspaceSelector = () => {
  const { currentWorkspace, workspaces, workspacesLoading, workspacesError, setCurrentWorkspace, refreshWorkspaces } = useAuth()
  const router = useRouter()
  const [search, setSearch] = useState("")

  const hasWorkspaces = workspaces.length > 0
  const showSearch = workspaces.length > SEARCH_THRESHOLD

  const filteredWorkspaces = useMemo(() => {
    if (!search) return workspaces
    const q = search.toLowerCase()
    return workspaces.filter(
      (ws) => ws.name.toLowerCase().includes(q) || ws.slug.toLowerCase().includes(q)
    )
  }, [workspaces, search])

  // Reset search when dropdown closes
  const handleOpenChange = (open: boolean) => {
    if (!open) setSearch("")
  }

  // ─── No workspaces: pulsing CTA trigger ───
  const noWorkspacesIdle = !workspacesLoading && !workspacesError && !hasWorkspaces

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu onOpenChange={handleOpenChange}>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                className={`w-full data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground h-auto ${
                  noWorkspacesIdle ? "animate-pulse border border-dashed border-green-500/60" : ""
                }`}
                tooltip="Workspace"
                aria-label={`Current workspace: ${currentWorkspace?.name ?? "None selected"}`}
              />
            }
          >
            {workspacesLoading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : noWorkspacesIdle ? (
              <Plus className="size-4 text-green-500" />
            ) : (
              <Building2 className="size-4" />
            )}
            <div className="flex flex-1 flex-col text-left text-sm leading-tight">
              <span className="truncate font-medium">
                {workspacesLoading
                  ? "Loading..."
                  : noWorkspacesIdle
                    ? "Create Workspace"
                    : currentWorkspace?.name ?? "Select Workspace"}
              </span>
              {currentWorkspace?.slug && !workspacesLoading && (
                <span className="truncate text-xs text-muted-foreground">
                  {currentWorkspace.slug}
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
            {/* Search input (only when >5 workspaces) */}
            {showSearch && (
              <div className="px-1.5 pb-1.5 pt-0.5">
                <Input
                  placeholder="Search workspaces..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="h-7 text-xs"
                  aria-label="Filter workspaces"
                  autoFocus
                />
              </div>
            )}

            <DropdownMenuGroup>
              <DropdownMenuLabel>Workspaces</DropdownMenuLabel>

              {/* Error state */}
              {workspacesError && (
                <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-destructive">
                  <AlertCircle className="size-3 shrink-0" />
                  <span className="truncate">{workspacesError}</span>
                  <Button
                    variant="link"
                    size="xs"
                    className="ml-auto shrink-0 text-xs"
                    onClick={() => refreshWorkspaces()}
                  >
                    Retry
                  </Button>
                </div>
              )}

              {/* Loading state */}
              {workspacesLoading && (
                <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-muted-foreground">
                  <Loader2 className="size-3 animate-spin" />
                  Loading workspaces...
                </div>
              )}

              {/* Workspace list with radio-style checkmarks */}
              {!workspacesLoading && !workspacesError && filteredWorkspaces.map((ws) => (
                <DropdownMenuItem
                  key={ws.id}
                  onClick={() => {
                    setCurrentWorkspace(ws)
                    router.refresh()
                  }}
                >
                  <Building2 className="mr-2 size-4 shrink-0" />
                  <span className="truncate">{ws.name}</span>
                  {currentWorkspace?.id === ws.id && (
                    <Check className="ml-auto size-4 shrink-0 text-muted-foreground" />
                  )}
                </DropdownMenuItem>
              ))}

              {/* Search yielded no results */}
              {!workspacesLoading && !workspacesError && hasWorkspaces && filteredWorkspaces.length === 0 && (
                <div className="px-2 py-1.5 text-xs text-muted-foreground">
                  No workspaces match &ldquo;{search}&rdquo;
                </div>
              )}

              {/* Empty state (no workspaces at all) */}
              {noWorkspacesIdle && (
                <div className="px-2 py-1.5 text-xs text-muted-foreground">
                  No workspaces yet
                </div>
              )}
            </DropdownMenuGroup>

            <DropdownMenuSeparator />

            {/* Action links */}
            <DropdownMenuItem
              onClick={() => router.push("/workspaces?create=true")}
            >
              <Plus className="mr-2 size-4" />
              Create Workspace
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => router.push("/workspaces")}
            >
              <Settings className="mr-2 size-4" />
              Manage Workspaces
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
