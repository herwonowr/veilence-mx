"use client"

import { useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { useAuth } from "@/core"
import { workspaceUpdateSchema, invitationSchema } from "@/domains/admin"
import { Button, Input, Field, FieldLabel, FieldError, Tabs, TabsContent, TabsList, TabsTrigger, Badge, Skeleton, ConfirmDialog, Alert, AlertDescription, TableEmptyState, type ConfirmDialogDetail } from "@/ui"
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import {
  ArrowLeft,
  Loader2,
  Save,
  Trash2,
  UserPlus,
  UserMinus,
  ShieldCheck,
  KeyRound,
  ScrollText,
  Mail,
} from "lucide-react"
import Link from "next/link"
import { ZodError } from "zod"
import {
  useWorkspace,
  useWorkspaceMembers,
  useWorkspaceRoles,
  useUpdateWorkspace,
  useDeleteWorkspace,
  useInviteMember,
  useRemoveMember,
  useUpdateMemberRole,
  usePendingInvitations,
  useRevokeInvitation,
  useResendInvitation,
} from "@/features/admin/hooks/use-workspaces"
import { useCurrentWorkspaceRole, hasMinimumRole } from "@/core"

export const WorkspaceDetailView = () => {
  const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)
  const params = useParams<{ id: string }>()
  const workspaceId = params.id
  const validWorkspaceId = workspaceId ?? ""
  const router = useRouter()
  const { user, refreshWorkspaces } = useAuth()

  const { data: workspaceRes, isLoading: workspaceLoading } = useWorkspace(validWorkspaceId)
  const { data: membersRes } = useWorkspaceMembers(validWorkspaceId)
  const { data: rolesRes } = useWorkspaceRoles(validWorkspaceId)
  const { data: invitationsRes } = usePendingInvitations(validWorkspaceId)

  const workspace = workspaceRes?.data ?? null
  const members = membersRes?.data ?? []
  const roles = rolesRes?.data ?? []
  const invitations = invitationsRes?.data ?? []

  // SEC-S3-007: Determine current user's permissions in this workspace
  // Use both permission-based checks (from member data) and role hierarchy
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

  // React-recommended "store previous props" pattern for syncing derived state
  if (workspace && prevWorkspaceId !== workspace.id) {
    setPrevWorkspaceId(workspace.id)
    setEditName(workspace.name)
    setEditDescription(workspace.description ?? "")
  }

  // Invite form
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState("")
  const [inviteRoleId, setInviteRoleId] = useState<string | null>(null)
  const [inviteError, setInviteError] = useState("")
  const [inviteFieldErrors, setInviteFieldErrors] = useState<Record<string, string>>({})

  // Edit field errors
  const [editFieldErrors, setEditFieldErrors] = useState<Record<string, string>>({})

  // Remove member confirmation
  const [memberToRemove, setMemberToRemove] = useState<{
    userId: string
    name: string
    details: ConfirmDialogDetail[]
  } | null>(null)

  const updateMutation = useUpdateWorkspace()
  const deleteMutation = useDeleteWorkspace()
  const inviteMutation = useInviteMember()
  const removeMutation = useRemoveMember()
  const updateRoleMutation = useUpdateMemberRole()
  const revokeMutation = useRevokeInvitation()
  const resendMutation = useResendInvitation()

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
    router.push("/workspaces")
  }

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    setInviteError("")
    setInviteFieldErrors({})
    try {
      invitationSchema.parse({ email: inviteEmail, roleId: inviteRoleId ? Number(inviteRoleId) : undefined })
    } catch (err) {
      if (err instanceof ZodError) {
        const errs: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") errs[key] = issue.message
        }
        setInviteFieldErrors(errs)
      }
      return
    }
    if (!inviteRoleId) return
    try {
      await inviteMutation.mutateAsync({
        workspaceId: validWorkspaceId,
        data: { email: inviteEmail, roleId: inviteRoleId },
      })
      setInviteDialogOpen(false)
      setInviteEmail("")
      setInviteRoleId(null)
    } catch (err) {
      setInviteError(
        err instanceof Error ? err.message : "Failed to send invitation"
      )
    }
  }

  const handleRemoveMember = async (userId: string) => {
    await removeMutation.mutateAsync({ workspaceId: validWorkspaceId, userId })
    setMemberToRemove(null)
  }

  const confirmRemoveMember = (userId: string, firstName?: string, lastName?: string, email?: string) => {
    const name = [firstName, lastName].filter(Boolean).join(" ") || "this member"
    setMemberToRemove({
      userId,
      name,
      details: [
        { label: "Member", value: name },
        ...(email ? [{ label: "Email", value: email }] : []),
        { label: "Workspace", value: workspace?.name ?? "" },
      ],
    })
  }

  const handleUpdateRole = (userId: string, roleId: string) => {
    updateRoleMutation.mutate({ workspaceId: validWorkspaceId, userId, roleId })
  }

  if (!workspaceId) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-lg font-medium text-destructive">Invalid workspace ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/workspaces")}>
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
        <Button variant="link" onClick={() => router.push("/workspaces")}>
          Back to Workspaces
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/workspaces"
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
            <Link href={`/workspaces/${workspace.id}/audit`}>
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
            {canInvite && (
            <Dialog
              open={inviteDialogOpen}
              onOpenChange={(open) => setInviteDialogOpen(open)}
            >
              <DialogTrigger
                render={
                  <Button size="sm">
                    <UserPlus className="mr-2 size-4" />
                    Invite Member
                  </Button>
                }
              />
              <DialogContent className="sm:max-w-md">
                <form onSubmit={handleInvite}>
                  <DialogHeader>
                    <DialogTitle>Invite Member</DialogTitle>
                    <DialogDescription>
                      Send an invitation to join this workspace.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    {inviteError && (
                      <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                        <AlertDescription>{inviteError}</AlertDescription>
                      </Alert>
                    )}
                    <Field data-invalid={!!inviteFieldErrors.email}>
                      <FieldLabel htmlFor="invite-email">Email</FieldLabel>
                      <Input
                        id="invite-email"
                        type="email"
                        placeholder="user@example.com"
                        value={inviteEmail}
                        onChange={(e) => setInviteEmail(e.target.value)}
                        required
                      />
                      {inviteFieldErrors.email && <FieldError>{inviteFieldErrors.email}</FieldError>}
                    </Field>
                    <Field data-invalid={!!inviteFieldErrors.roleId}>
                      <FieldLabel>Role</FieldLabel>
                      <Select
                        value={inviteRoleId != null ? String(inviteRoleId) : undefined}
                        onValueChange={(v) => setInviteRoleId(v || null)}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue>{inviteRoleId != null ? capitalize(roles.find(r => r.id === inviteRoleId)?.name ?? "") : "Select a role"}</SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          {roles.filter((role) => role.name.toLowerCase() !== "owner").map((role) => (
                            <SelectItem key={role.id} value={String(role.id)}>
                              {capitalize(role.name)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {inviteFieldErrors.roleId && <FieldError>{inviteFieldErrors.roleId}</FieldError>}
                    </Field>
                  </div>
                  <DialogFooter>
                    <Button type="submit" disabled={inviteMutation.isPending || !inviteRoleId}>
                      {inviteMutation.isPending && (
                        <Loader2 className="mr-2 size-4 animate-spin" />
                      )}
                      Send Invitation
                    </Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
            )}
          </div>

          <Card>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Joined</TableHead>
                    <TableHead className="w-[1%] whitespace-nowrap text-right">
                      <span className="sr-only">Actions</span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {members.map((member) => (
                    <TableRow key={member.id}>
                      <TableCell>
                        <div>
                          <p className="font-medium">
                            {member.firstName} {member.lastName}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {member.email}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>
                        {member.userId === workspace.ownerId || !canUpdateRole ? (
                          <Badge variant="secondary">
                            {member.role?.name ? capitalize(member.role.name) : member.userId === workspace.ownerId ? "Owner" : "No role"}
                          </Badge>
                        ) : (
                          <Select
                            value={String(member.roleId)}
                            onValueChange={(v) =>
                              handleUpdateRole(member.userId, String(v))
                            }
                            disabled={member.userId === user?.id}
                          >
                            <SelectTrigger className="w-28">
                              <SelectValue>{member.role?.name ? capitalize(member.role.name) : "No role"}</SelectValue>
                            </SelectTrigger>
                            <SelectContent>
                              {roles.filter((role) => role.name.toLowerCase() !== "owner").map((role) => (
                                <SelectItem key={role.id} value={String(role.id)}>
                                  {capitalize(role.name)}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(member.joinedAt).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}
                      </TableCell>
                      <TableCell className="text-right">
                        {member.userId !== user?.id && canRemove && (
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Remove member ${member.firstName} ${member.lastName}`}
                            onClick={() =>
                              confirmRemoveMember(
                                member.userId,
                                member.firstName,
                                member.lastName,
                                member.email
                              )
                            }
                          >
                            <UserMinus className="size-4 text-destructive" />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {members.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={4}
                        className="text-center py-8 text-muted-foreground"
                      >
                        No members found
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* Invitations - visible to all roles, actions gated by canManageMembers */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Mail className="size-4" />
              Invitations ({invitations.length})
            </h3>
            <Card>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Email</TableHead>
                      <TableHead>Role</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Invited</TableHead>
                      <TableHead>Expires</TableHead>
                      {canManageMembers && (
                      <TableHead className="w-[1%] whitespace-nowrap text-right">
                        <span className="sr-only">Actions</span>
                      </TableHead>
                      )}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {invitations.length === 0 && (
                      <TableEmptyState
                        colSpan={canManageMembers ? 6 : 5}
                        icon={<Mail className="size-8" />}
                        title="No invitations"
                        description={canManageMembers ? "Invite members to join this workspace." : "No pending invitations for this workspace."}
                      />
                    )}
                    {invitations.map((invitation) => {
                      const role = roles.find((r) => r.id === invitation.roleId)
                      const status = invitation.status
                      const statusVariant = status === "accepted"
                        ? "default"
                        : status === "expired"
                          ? "destructive"
                          : "secondary"
                      const dateOptions: Intl.DateTimeFormatOptions = {
                        year: "numeric",
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      }
                      return (
                        <TableRow key={invitation.id}>
                          <TableCell>
                            <span className="font-medium">{invitation.email}</span>
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary">
                              {role ? capitalize(role.name) : "Unknown"}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant={statusVariant}>
                              {capitalize(status)}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {new Date(invitation.createdAt).toLocaleDateString("en-US", dateOptions)}
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {new Date(invitation.expiresAt).toLocaleDateString("en-US", dateOptions)}
                          </TableCell>
                          {canManageMembers && (
                          <TableCell className="text-right">
                            {canInvite && status === "pending" && (
                              <div className="flex items-center justify-end gap-2">
                                <Button
                                  variant="outline"
                                  size="xs"
                                  aria-label={`Resend invitation for ${invitation.email}`}
                                  disabled={resendMutation.isPending}
                                  onClick={() =>
                                    resendMutation.mutate({
                                      workspaceId: validWorkspaceId,
                                      invitationId: invitation.id,
                                    })
                                  }
                                >
                                  {resendMutation.isPending ? (
                                    <Loader2 className="mr-1 size-3 animate-spin" />
                                  ) : null}
                                  Resend
                                </Button>
                                <Button
                                  variant="destructive"
                                  size="xs"
                                  aria-label={`Revoke invitation for ${invitation.email}`}
                                  disabled={revokeMutation.isPending}
                                  onClick={() =>
                                    revokeMutation.mutate({
                                      workspaceId: validWorkspaceId,
                                      invitationId: invitation.id,
                                    })
                                  }
                                >
                                  Revoke
                                </Button>
                              </div>
                            )}
                          </TableCell>
                          )}
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>

          {/* Remove Member Confirmation Dialog */}
          <ConfirmDialog
            open={!!memberToRemove}
            onOpenChange={(open) => { if (!open) setMemberToRemove(null) }}
            title="Remove Member?"
            description={`Are you sure you want to remove ${memberToRemove?.name ?? "this member"} from this workspace? They will lose access to all workspace resources.`}
            details={memberToRemove?.details}
            actionLabel="Remove"
            onConfirm={async () => {
              if (memberToRemove) {
                await handleRemoveMember(memberToRemove.userId)
              }
            }}
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
