"use client"

import { useCallback, useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useAuth, ROUTES, usePublicConfigQuery, useDebouncedValue } from "@/core"
import { apiCreateWorkspace, workspaceSchema } from "@/domains/admin"
import type { Workspace } from "@/domains/admin"
import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldDescription,
  FieldError,
  Badge,
  EmptyState,
  Alert,
  AlertDescription,
  SearchInput,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/ui"
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
import { Layers, Plus, Loader2, Users, Package, Mail, LayoutGrid, Table2, Search, ChevronLeft, ChevronRight } from "lucide-react"
import Link from "next/link"
import { useWorkspaces } from "@/features/admin/hooks/use-workspaces"
import { useMyInvitations } from "@/features/admin/hooks/use-my-invitations"
import { ZodError } from "zod"

type ViewMode = "grid" | "table"

const VIEW_MODE_KEY = "veilence-workspaces-view-mode"

const getStoredViewMode = (): ViewMode => {
  if (typeof window === "undefined") return "grid"
  const stored = localStorage.getItem(VIEW_MODE_KEY)
  return stored === "table" ? "table" : "grid"
}

export const WorkspacesListView = () => {
  const { refreshWorkspaces, setCurrentWorkspace } = useAuth()
  const { registrationEnabled } = usePublicConfigQuery()

  // Search and view mode state
  const [searchQuery, setSearchQuery] = useState("")
  const [viewMode, setViewMode] = useState<ViewMode>(getStoredViewMode)
  const [page, setPage] = useState(1)
  const limit = 20

  const debouncedSearch = useDebouncedValue(searchQuery, 300)

  const { data: workspacesRes } = useWorkspaces({
    search: debouncedSearch || undefined,
    page,
    limit,
  })
  const { data: myInvitationsRes } = useMyInvitations()
  const workspaces = workspacesRes?.data ?? []
  const meta = workspacesRes?.meta
  const totalPages = meta ? Math.ceil(meta.total / meta.limit) : 1
  const pendingInvitationCount = (myInvitationsRes?.data ?? []).filter(
    (inv) => inv.status === "pending"
  ).length
  const router = useRouter()
  const searchParams = useSearchParams()
  const shouldCreateWorkspace = searchParams.get("create") === "true"

  const [dialogOpenByUser, setDialogOpenByUser] = useState(shouldCreateWorkspace)

  if (shouldCreateWorkspace && !dialogOpenByUser) {
    setDialogOpenByUser(true)
  }

  const dialogOpen = dialogOpenByUser || shouldCreateWorkspace

  const setDialogOpen = useCallback((open: boolean) => {
    setDialogOpenByUser(open)
  }, [])

  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const [description, setDescription] = useState("")
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState("")
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const handleViewModeChange = useCallback((mode: ViewMode) => {
    setViewMode(mode)
    localStorage.setItem(VIEW_MODE_KEY, mode)
  }, [])

  useEffect(() => {
    if (shouldCreateWorkspace) {
      router.replace(ROUTES.WORKSPACES, { scroll: false })
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

  const handleCreate = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError("")
    setFieldErrors({})

    try {
      workspaceSchema.parse({ name, slug, description })
    } catch (err) {
      if (err instanceof ZodError) {
        const errs: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") errs[key] = issue.message
        }
        setFieldErrors(errs)
      }
      return
    }

    setCreating(true)

    try {
      const { data } = await apiCreateWorkspace({ name, slug, description })
      setCurrentWorkspace(data)
      await refreshWorkspaces()
      setDialogOpen(false)
      setName("")
      setSlug("")
      setDescription("")
      router.push(ROUTES.WORKSPACE_DETAIL(data.id))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create workspace")
    } finally {
      setCreating(false)
    }
  }

  const isSearching = debouncedSearch.length > 0
  const hasWorkspaces = workspaces.length > 0 || isSearching
  const hasResults = workspaces.length > 0

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Workspaces</h1>
        <div className="flex items-center gap-3">
          {registrationEnabled && (
            <Link href="/workspaces/invitations">
              <Button variant="outline">
                <Mail className="mr-2 size-4" />
                View Invitations
                {pendingInvitationCount > 0 && (
                  <Badge variant="destructive" className="ml-2">
                    {pendingInvitationCount}
                  </Badge>
                )}
              </Button>
            </Link>
          )}
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
                  <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <Field data-invalid={!!fieldErrors.name}>
                  <FieldLabel htmlFor="workspace-name">Name</FieldLabel>
                  <Input
                    id="workspace-name"
                    placeholder="Acme Corp"
                    value={name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    required
                  />
                  {fieldErrors.name && <FieldError>{fieldErrors.name}</FieldError>}
                </Field>
                <Field data-invalid={!!fieldErrors.slug}>
                  <FieldLabel htmlFor="workspace-slug">Slug</FieldLabel>
                  <Input
                    id="workspace-slug"
                    placeholder="acme-corp"
                    value={slug}
                    onChange={(e) => setSlug(e.target.value)}
                    required
                  />
                  {fieldErrors.slug && <FieldError>{fieldErrors.slug}</FieldError>}
                  <FieldDescription>
                    URL-friendly identifier for your workspace
                  </FieldDescription>
                </Field>
                <Field data-invalid={!!fieldErrors.description}>
                  <FieldLabel htmlFor="workspace-description">Description</FieldLabel>
                  <Input
                    id="workspace-description"
                    placeholder="Optional description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                  {fieldErrors.description && <FieldError>{fieldErrors.description}</FieldError>}
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
      </div>

      {/* Toolbar: search + view toggle */}
      {hasWorkspaces && (
        <div className="flex items-center justify-between gap-4">
          <SearchInput
            value={searchQuery}
            onChange={(value) => {
              setSearchQuery(value)
              setPage(1)
            }}
            placeholder="Search workspaces..."
            aria-label="Search workspaces by name or slug"
            className="max-w-sm"
          />
          <div className="flex items-center gap-1">
            <Button
              variant={viewMode === "grid" ? "secondary" : "ghost"}
              size="icon"
              onClick={() => handleViewModeChange("grid")}
              aria-label="Grid view"
              aria-pressed={viewMode === "grid"}
            >
              <LayoutGrid className="size-4" />
            </Button>
            <Button
              variant={viewMode === "table" ? "secondary" : "ghost"}
              size="icon"
              onClick={() => handleViewModeChange("table")}
              aria-label="Table view"
              aria-pressed={viewMode === "table"}
            >
              <Table2 className="size-4" />
            </Button>
          </div>
        </div>
      )}

      {!hasWorkspaces ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<Layers className="h-12 w-12" />}
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
      ) : !hasResults && isSearching ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<Search className="h-12 w-12" />}
              title="No workspaces found"
              description={`No workspaces match "${debouncedSearch}". Try a different search term.`}
            />
          </CardContent>
        </Card>
      ) : viewMode === "grid" ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workspaces.map((ws) => (
            <WorkspaceCard key={ws.id} workspace={ws} />
          ))}
        </div>
      ) : (
        <WorkspacesTable workspaces={workspaces} />
      )}

      {/* Pagination */}
      {hasResults && totalPages > 1 && (
        <div className="flex items-center justify-center gap-4">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            <ChevronLeft className="mr-1 size-4" />
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
            <ChevronRight className="ml-1 size-4" />
          </Button>
        </div>
      )}
    </div>
  )
}

const WorkspaceCard = ({ workspace }: { workspace: Workspace }) => {
  const memberCount = workspace.memberCount ?? null
  const packageCount = workspace.packageCount ?? null

  return (
    <Link href={`/workspaces/${workspace.id}`}>
      <Card className="cursor-pointer transition-colors hover:bg-muted/50">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">{workspace.name}</CardTitle>
            <Badge variant={["owner", "admin"].includes(workspace.role) ? "secondary" : "outline"}>
              {workspace.role.charAt(0).toUpperCase() + workspace.role.slice(1)}
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

const WorkspacesTable = ({ workspaces }: { workspaces: Workspace[] }) => {
  const router = useRouter()

  return (
    <Card>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Slug</TableHead>
            <TableHead>Members</TableHead>
            <TableHead>Packages</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Role</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {workspaces.map((ws) => (
            <TableRow
              key={ws.id}
              clickable
              onClick={() => router.push(`/workspaces/${ws.id}`)}
            >
              <TableCell className="font-medium">{ws.name}</TableCell>
              <TableCell className="font-mono text-xs text-muted-foreground">
                {ws.slug}
              </TableCell>
              <TableCell>
                <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Users className="size-3.5" />
                  {ws.memberCount ?? "-"}
                </span>
              </TableCell>
              <TableCell>
                <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Package className="size-3.5" />
                  {ws.packageCount ?? "-"}
                </span>
              </TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {new Date(ws.createdAt).toLocaleDateString()}
              </TableCell>
              <TableCell>
                <Badge variant={["owner", "admin"].includes(ws.role) ? "secondary" : "outline"}>
                  {ws.role.charAt(0).toUpperCase() + ws.role.slice(1)}
                </Badge>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Card>
  )
}
