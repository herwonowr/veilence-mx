"use client"

import { useCallback, useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useAuth } from "@/core"
import { apiCreateWorkspace } from "@/domains/admin"
import type { Workspace } from "@/domains/admin"
import { Button, Input, Field, FieldLabel, FieldDescription, Badge, EmptyState, Alert, AlertDescription } from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/ui"
import { Building2, Plus, Loader2, Users, Package } from "lucide-react"
import Link from "next/link"
import { useWorkspaceMembers } from "@/features/admin/hooks/use-workspaces"
import { useAdminPackages } from "@/features/admin/hooks/use-admin-packages"

export const WorkspacesListView = () => {
  const { workspaces, refreshWorkspaces, setCurrentWorkspace } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()
  const shouldCreateWorkspace = searchParams.get("create") === "true"

  // Dialog open state: initially true if URL has ?create=true, then controlled by user interaction.
  // The shouldCreateWorkspace value is read on mount via the state initializer;
  // subsequent navigations to ?create=true cause shouldCreateWorkspace to become true,
  // which we OR into the derived dialogOpen below.
  const [dialogOpenByUser, setDialogOpenByUser] = useState(shouldCreateWorkspace)
  const dialogOpen = dialogOpenByUser || shouldCreateWorkspace

  const setDialogOpen = useCallback((open: boolean) => {
    setDialogOpenByUser(open)
  }, [])

  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const [description, setDescription] = useState("")
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState("")

  // When URL has ?create=true, clean it up so re-navigation works
  useEffect(() => {
    if (shouldCreateWorkspace) {
      router.replace("/workspaces", { scroll: false })
    }
  }, [shouldCreateWorkspace, router])

  const generateSlug = useCallback((value: string) => {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "")
  }, [])

  const handleNameChange = (value: string) => {
    setName(value)
    setSlug(generateSlug(value))
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setCreating(true)

    try {
      const { data } = await apiCreateWorkspace({ name, slug, description })
      setCurrentWorkspace(data)
      await refreshWorkspaces()
      setDialogOpen(false)
      setName("")
      setSlug("")
      setDescription("")
      router.push(`/workspaces/${data.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create workspace")
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Workspaces</h1>
        <Dialog open={dialogOpen} onOpenChange={(open) => setDialogOpen(open)}>
          <DialogTrigger
            render={
              <Button>
                <Plus className="mr-2 size-4" />
                New Workspace
              </Button>
            }
          />
          <DialogContent className="sm:max-w-md">
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Create Workspace</DialogTitle>
                <DialogDescription>
                  Create a new workspace to manage packages and team members.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                {error && (
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <Field>
                  <FieldLabel htmlFor="workspace-name">Name</FieldLabel>
                  <Input
                    id="workspace-name"
                    placeholder="My Workspace"
                    value={name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    required
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="workspace-slug">Slug</FieldLabel>
                  <Input
                    id="workspace-slug"
                    placeholder="my-workspace"
                    value={slug}
                    onChange={(e) => setSlug(e.target.value)}
                    required
                  />
                  <FieldDescription>
                    URL-friendly identifier for your workspace
                  </FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor="workspace-description">Description</FieldLabel>
                  <Input
                    id="workspace-description"
                    placeholder="Optional description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </Field>
              </div>
              <DialogFooter>
                <Button type="submit" disabled={creating}>
                  {creating && (
                    <Loader2 className="mr-2 size-4 animate-spin" />
                  )}
                  Create
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {workspaces.length === 0 ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<Building2 className="h-12 w-12" />}
              title="No workspaces yet"
              description="Create your first workspace to get started."
            >
              <Button onClick={() => setDialogOpen(true)}>
                <Plus className="mr-2 size-4" />
                Create Workspace
              </Button>
            </EmptyState>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workspaces.map((ws: Workspace) => (
            <WorkspaceCard key={ws.id} workspace={ws} />
          ))}
        </div>
      )}
    </div>
  )
}

const WorkspaceCard = ({ workspace }: { workspace: Workspace }) => {
  const { data: membersRes } = useWorkspaceMembers(workspace.id)
  const { data: packagesRes } = useAdminPackages({ page: 1, limit: 1 })

  const memberCount = membersRes?.data?.length ?? null
  const packageCount = packagesRes?.meta?.total ?? null

  return (
    <Link href={`/workspaces/${workspace.id}`}>
      <Card className="cursor-pointer transition-colors hover:bg-muted/50">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">{workspace.name}</CardTitle>
            <Badge variant={workspace.isActive ? "secondary" : "outline"}>
              {workspace.isActive ? "Active" : "Inactive"}
            </Badge>
          </div>
          <CardDescription className="font-mono text-xs">
            {workspace.slug}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground line-clamp-2">
            {workspace.description || "No description"}
          </p>
          <div className="flex items-center gap-4 mt-3">
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Users className="size-3.5" />
              {memberCount !== null ? memberCount : "-"} {memberCount === 1 ? "member" : "members"}
            </span>
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Package className="size-3.5" />
              {packageCount !== null ? packageCount : "-"} {packageCount === 1 ? "package" : "packages"}
            </span>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            Created {new Date(workspace.createdAt).toLocaleDateString()}
          </p>
        </CardContent>
      </Card>
    </Link>
  )
}
