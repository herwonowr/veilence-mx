"use client"

import { useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { useAuth, ROUTES, useCurrentWorkspaceRole, hasMinimumRole , usePublicConfigQuery } from "@/core"
import { workspaceUpdateSchema } from "@/domains/admin"
import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldError,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Badge,
  Skeleton,
  ConfirmDialog,
} from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import {
  ArrowLeft,
  Loader2,
  Save,
  Trash2,
  ShieldCheck,
  KeyRound,
  ScrollText,
} from "lucide-react"
import Link from "next/link"
import { ZodError } from "zod"
import {
  useWorkspace,
  useWorkspaceMembers,
  useWorkspaceRoles,
  useUpdateWorkspace,
  useDeleteWorkspace,
  usePendingInvitations,
} from "@/features/admin"
import { AddMemberDialog } from "@/features/admin/ui/add-member-dialog"
import { InviteMemberDialog } from "@/features/admin/ui/invite-member-dialog"
import { MembersSection } from "@/features/admin/ui/members-section"

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

export const WorkspaceDetailView = () => {
  const params = useParams<{ id: string }>()
  const workspaceId = params.id
  const validWorkspaceId = workspaceId ?? ""
  const router = useRouter()
  const { user, refreshWorkspaces } = useAuth()

  const { data: workspaceRes, isLoading: workspaceLoading } = useWorkspace(validWorkspaceId)
  const { data: membersRes } = useWorkspaceMembers(validWorkspaceId)
  const { data: rolesRes } = useWorkspaceRoles(validWorkspaceId)
  const { data: invitationsRes } = usePendingInvitations(validWorkspaceId)

  const { registrationEnabled } = usePublicConfigQuery()

  const workspace = workspaceRes?.data ?? null
  const members = membersRes?.data ?? []
  const roles = rolesRes?.data ?? []
  const invitations = invitationsRes?.data ?? []

  const { role: currentRole } = useCurrentWorkspaceRole()
  const currentMember = members.find((m) => m.userId === user?.id)
  const currentPermissions = currentMember?.role?.permissions ?? []
  const hasPermission = (resource: string, action: string) =>
    currentPermissions.some(
      (p) => p.resource === resource && p.action === action
    )
  const isWorkspaceOwner = workspace?.ownerId === user?.id
  const isAdminOrAbove = hasMinimumRole(currentRole, "admin")
  const canInvite = isWorkspaceOwner || hasPermission("members", "invite")
  const canRemove = isWorkspaceOwner || hasPermission("members", "remove")
  const canUpdateRole = isWorkspaceOwner || hasPermission("members", "update_role")
  const canManageMembers = isAdminOrAbove || isWorkspaceOwner
  const canUpdateWorkspace = isWorkspaceOwner
  const canViewAuditLog = isAdminOrAbove || isWorkspaceOwner

  // Edit form
  const [editName, setEditName] = useState("")
  const [editDescription, setEditDescription] = useState("")
  const [prevWorkspaceId, setPrevWorkspaceId] = useState<string | null>(null)
  const [editFieldErrors, setEditFieldErrors] = useState<Record<string, string>>({})

  // React-recommended "store previous props" pattern for syncing derived state
  if (workspace && prevWorkspaceId !== workspace.id) {
    setPrevWorkspaceId(workspace.id)
    setEditName(workspace.name)
    setEditDescription(workspace.description ?? "")
  }

  const updateMutation = useUpdateWorkspace()
  const deleteMutation = useDeleteWorkspace()

  const handleSave = async () => {
    setEditFieldErrors({})
    try {
      workspaceUpdateSchema.parse({ name: editName, description: editDescription })
    } catch (err) {
      if (err instanceof ZodError) {
        const errs: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") errs[key] = issue.message
        }
        setEditFieldErrors(errs)
      }
      return
    }
    await updateMutation.mutateAsync({
      id: validWorkspaceId,
      data: { name: editName, description: editDescription },
    })
    await refreshWorkspaces()
  }

  const handleDelete = async () => {
    await deleteMutation.mutateAsync(validWorkspaceId)
    await refreshWorkspaces()
    router.push(ROUTES.WORKSPACES)
  }

  if (!workspaceId) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-lg font-medium text-destructive">Invalid workspace ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push(ROUTES.WORKSPACES)}>
          Back to Workspaces
        </Button>
      </div>
    )
  }

  if (workspaceLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!workspace) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-medium">Workspace not found</h2>
        <Button variant="link" onClick={() => router.push(ROUTES.WORKSPACES)}>
          Back to Workspaces
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href={ROUTES.WORKSPACES}
          className="w-fit text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Workspaces
        </Link>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">{workspace.name}</h1>
            <p className="text-sm text-muted-foreground font-mono">{workspace.slug}</p>
          </div>
          <div className="flex items-center gap-2">
            {canViewAuditLog && (
            <Link href={ROUTES.WORKSPACE_AUDIT(workspace.id)}>
              <Button variant="outline" size="sm">
                <ScrollText className="mr-2 size-4" />
                Audit Log
              </Button>
            </Link>
            )}
          </div>
        </div>
      </div>

      <Tabs defaultValue="members">
        <TabsList>
          <TabsTrigger value="members">Members</TabsTrigger>
          <TabsTrigger value="roles">Roles</TabsTrigger>
          {canUpdateWorkspace && (
            <TabsTrigger value="settings">Settings</TabsTrigger>
          )}
        </TabsList>

        {/* Members Tab */}
        <TabsContent value="members" className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-medium">Team Members</h2>
            <div className="flex items-center gap-2">
              {canInvite && (
                <AddMemberDialog
                  workspaceId={validWorkspaceId}
                  roles={roles}
                  capitalize={capitalize}
                />
              )}
              {canInvite && registrationEnabled && (
                <InviteMemberDialog
                  workspaceId={validWorkspaceId}
                  roles={roles}
                  capitalize={capitalize}
                />
              )}
            </div>
          </div>

          <MembersSection
            workspaceId={validWorkspaceId}
            workspace={workspace}
            members={members}
            roles={roles}
            invitations={invitations}
            currentUserId={user?.id}
            canInvite={canInvite}
            canRemove={canRemove}
            canUpdateRole={canUpdateRole}
            canManageMembers={canManageMembers}
            registrationEnabled={registrationEnabled}
            capitalize={capitalize}
          />
        </TabsContent>

        {/* Roles Tab */}
        <TabsContent value="roles" className="space-y-4">
          <h2 className="text-lg font-medium">Roles & Permissions</h2>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {roles.map((role) => (
              <Card key={role.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base flex items-center gap-2">
                      <ShieldCheck className="size-4" />
                      {capitalize(role.name)}
                    </CardTitle>
                    {role.isSystem && (
                      <Badge variant="secondary">System</Badge>
                    )}
                  </div>
                  <CardDescription>{role.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-1">
                    {role.permissions?.map((perm) => (
                      <Badge key={perm.id} variant="outline" className="text-xs">
                        <KeyRound className="mr-1 size-3" />
                        {perm.resource}:{perm.action}
                      </Badge>
                    ))}
                    {(!role.permissions || role.permissions.length === 0) && (
                      <span className="text-xs text-muted-foreground">
                        No permissions assigned
                      </span>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Settings Tab - owner only */}
        {canUpdateWorkspace && (
        <TabsContent value="settings" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Workspace Settings</CardTitle>
              <CardDescription>
                Update your workspace&apos;s name and description.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Field data-invalid={!!editFieldErrors.name}>
                <FieldLabel htmlFor="edit-name">Name</FieldLabel>
                <Input
                  id="edit-name"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
                {editFieldErrors.name && <FieldError>{editFieldErrors.name}</FieldError>}
              </Field>
              <Field data-invalid={!!editFieldErrors.description}>
                <FieldLabel htmlFor="edit-description">Description</FieldLabel>
                <Input
                  id="edit-description"
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                />
                {editFieldErrors.description && <FieldError>{editFieldErrors.description}</FieldError>}
              </Field>
              <div className="flex items-center gap-4">
                <Button onClick={handleSave} disabled={updateMutation.isPending}>
                  {updateMutation.isPending ? (
                    <Loader2 className="mr-2 size-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 size-4" />
                  )}
                  Save Changes
                </Button>
                {updateMutation.isSuccess && (
                  <span className="text-sm text-green-600">
                    Saved successfully
                  </span>
                )}
              </div>
            </CardContent>
          </Card>

          <Card className="border-destructive/50">
            <CardHeader>
              <CardTitle className="text-destructive">Danger Zone</CardTitle>
              <CardDescription>
                Permanently delete this workspace and all its data.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ConfirmDialog
                title="Delete Workspace?"
                description="Are you sure? This action cannot be undone. All data associated with this workspace will be permanently deleted."
                details={[
                  { label: "Workspace", value: workspace.name },
                  { label: "Slug", value: workspace.slug },
                  { label: "Members", value: String(members.length) },
                ]}
                actionLabel="Delete"
                onConfirm={handleDelete}
              >
                <Button variant="destructive">
                  <Trash2 className="mr-2 size-4" />
                  Delete Workspace
                </Button>
              </ConfirmDialog>
            </CardContent>
          </Card>
        </TabsContent>
        )}
      </Tabs>
    </div>
  )
}
