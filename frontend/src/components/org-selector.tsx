"use client"

import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { Building2, ChevronsUpDown, Plus } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
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
  const { currentOrg, organizations, setCurrentOrg } = useAuth()
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
              />
            }
          >
            <Building2 className="size-4" />
            <div className="flex flex-1 flex-col text-left text-sm leading-tight">
              <span className="truncate font-medium">
                {currentOrg?.name ?? "No Organization"}
              </span>
              {currentOrg?.slug && (
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
            <DropdownMenuLabel>Organizations</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {organizations.map((org) => (
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
            {organizations.length === 0 && (
              <div className="px-2 py-1.5 text-xs text-muted-foreground">
                No organizations yet
              </div>
            )}
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
