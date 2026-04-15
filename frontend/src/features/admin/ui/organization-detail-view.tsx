"use client"

import { useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { useAuth } from "@/core/providers/auth-provider"
import { Button } from "@/ui/components/button"
import { Input } from "@/ui/components/input"
import { Field, FieldLabel } from "@/ui/components/field"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui/components/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/ui/components/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui/components/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/ui/components/tabs"
import { Badge } from "@/ui/components/badge"
import { Skeleton } from "@/ui/components/skeleton"
import { ConfirmDialog, type ConfirmDialogDetail } from "@/ui/feedback/confirm-dialog"
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
} from "lucide-react"
import { Alert, AlertDescription } from "@/ui/components/alert"
import Link from "next/link"
import {
  useOrganization,
  useOrgMembers,
  useOrgRoles,
  useUpdateOrganization,
  useDeleteOrganization,
  useInviteMember,
  useRemoveMember,
  useUpdateMemberRole,
} from "@/features/admin/hooks/use-organizations"

export const OrganizationDetailView = () => {
  const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)
  const params = useParams<{ id: string }>()
  const orgId = parseInt(params.id, 10)
  const validOrgId = isNaN(orgId) ? 0 : orgId
  const router = useRouter()
  const { user, refreshOrgs } = useAuth()

  const { data: orgRes, isLoading: orgLoading } = useOrganization(validOrgId)
  const { data: membersRes } = useOrgMembers(validOrgId)
  const { data: rolesRes } = useOrgRoles(validOrgId)

  const org = orgRes?.data ?? null
  const members = membersRes?.data ?? []
  const roles = rolesRes?.data ?? []

  // SEC-S3-007: Determine current user's permissions in this org
  const currentMember = members.find((m) => m.userId === user?.id)
  const currentPermissions = currentMember?.role?.permissions ?? []
  const hasPermission = (resource: string, action: string) =>
    currentPermissions.some(
      (p) => p.resource === resource && p.action === action
    )
  const isOrgOwner = org?.ownerId === user?.id
  const canInvite = isOrgOwner || hasPermission("members", "invite")
  const canRemove = isOrgOwner || hasPermission("members", "remove")
  const canUpdateRole = isOrgOwner || hasPermission("members", "update_role")
  const canUpdateOrg = isOrgOwner

  // Edit form
  const [editName, setEditName] = useState("")
  const [editDescription, setEditDescription] = useState("")
  const [prevOrgId, setPrevOrgId] = useState<number | null>(null)

  // React-recommended "store previous props" pattern for syncing derived state
  if (org && prevOrgId !== org.id) {
    setPrevOrgId(org.id)
    setEditName(org.name)
    setEditDescription(org.description ?? "")
  }

  // Invite form
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState("")
  const [inviteRoleId, setInviteRoleId] = useState<number | null>(null)
  const [inviteError, setInviteError] = useState("")

  // Remove member confirmation
  const [memberToRemove, setMemberToRemove] = useState<{
    userId: number
    name: string
    details: ConfirmDialogDetail[]
  } | null>(null)

  const updateMutation = useUpdateOrganization()
  const deleteMutation = useDeleteOrganization()
  const inviteMutation = useInviteMember()
  const removeMutation = useRemoveMember()
  const updateRoleMutation = useUpdateMemberRole()

  const handleSave = async () => {
    await updateMutation.mutateAsync({
      id: validOrgId,
      data: { name: editName, description: editDescription },
    })
    await refreshOrgs()
  }

  const handleDelete = async () => {
    await deleteMutation.mutateAsync(validOrgId)
    await refreshOrgs()
    router.push("/organizations")
  }

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!inviteRoleId) return
    setInviteError("")
    try {
      await inviteMutation.mutateAsync({
        orgId: validOrgId,
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

  const handleRemoveMember = async (userId: number) => {
    await removeMutation.mutateAsync({ orgId: validOrgId, userId })
    setMemberToRemove(null)
  }

  const confirmRemoveMember = (userId: number, firstName?: string, lastName?: string, email?: string) => {
    const name = [firstName, lastName].filter(Boolean).join(" ") || "this member"
    setMemberToRemove({
      userId,
      name,
      details: [
        { label: "Member", value: name },
        ...(email ? [{ label: "Email", value: email }] : []),
        { label: "Organization", value: org?.name ?? "" },
      ],
    })
  }

  const handleUpdateRole = (userId: number, roleId: number) => {
    updateRoleMutation.mutate({ orgId: validOrgId, userId, roleId })
  }

  if (isNaN(orgId)) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-lg font-medium text-destructive">Invalid organization ID</p>
        <Button variant="outline" className="mt-4" onClick={() => router.push("/organizations")}>
          Back to Organizations
        </Button>
      </div>
    )
  }

  if (orgLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!org) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-medium">Organization not found</h2>
        <Button variant="link" onClick={() => router.push("/organizations")}>
          Back to Organizations
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/organizations"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Organizations
        </Link>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">{org.name}</h1>
            <p className="text-sm text-muted-foreground font-mono">{org.slug}</p>
          </div>
          <div className="flex items-center gap-2">
            <Link href={`/organizations/${org.id}/audit`}>
              <Button variant="outline" size="sm">
                <ScrollText className="mr-2 size-4" />
                Audit Log
              </Button>
            </Link>
          </div>
        </div>
      </div>

      <Tabs defaultValue="members">
        <TabsList>
          <TabsTrigger value="members">Members</TabsTrigger>
          <TabsTrigger value="roles">Roles</TabsTrigger>
          {canUpdateOrg && (
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
                      Send an invitation to join this organization.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    {inviteError && (
                      <Alert variant="destructive">
                        <AlertDescription>{inviteError}</AlertDescription>
                      </Alert>
                    )}
                    <Field>
                      <FieldLabel htmlFor="invite-email">Email</FieldLabel>
                      <Input
                        id="invite-email"
                        type="email"
                        placeholder="user@example.com"
                        value={inviteEmail}
                        onChange={(e) => setInviteEmail(e.target.value)}
                        required
                      />
                    </Field>
                    <Field>
                      <FieldLabel>Role</FieldLabel>
                      <Select
                        value={inviteRoleId != null ? String(inviteRoleId) : undefined}
                        onValueChange={(v) => setInviteRoleId(v ? parseInt(String(v), 10) : null)}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue>{inviteRoleId != null ? capitalize(roles.find(r => r.id === inviteRoleId)?.name ?? "") : "Select a role"}</SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          {roles.map((role) => (
                            <SelectItem key={role.id} value={String(role.id)}>
                              {capitalize(role.name)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
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
            <CardContent className="p-0">
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
                        <Select
                          value={String(member.roleId)}
                          onValueChange={(v) =>
                            handleUpdateRole(member.userId, parseInt(String(v), 10))
                          }
                          disabled={member.userId === user?.id || !canUpdateRole}
                        >
                          <SelectTrigger className="w-28">
                            <SelectValue>{member.role?.name ? capitalize(member.role.name) : "..."}</SelectValue>
                          </SelectTrigger>
                          <SelectContent>
                            {roles.map((role) => (
                              <SelectItem key={role.id} value={String(role.id)}>
                                {capitalize(role.name)}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(member.joinedAt).toLocaleDateString()}
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

          {/* Remove Member Confirmation Dialog */}
          <ConfirmDialog
            open={!!memberToRemove}
            onOpenChange={(open) => { if (!open) setMemberToRemove(null) }}
            title="Remove Member?"
            description={`Are you sure you want to remove ${memberToRemove?.name ?? "this member"} from this organization? They will lose access to all organization resources.`}
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

        {/* Settings Tab — owner only */}
        {canUpdateOrg && (
        <TabsContent value="settings" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Organization Settings</CardTitle>
              <CardDescription>
                Update your organization&apos;s name and description.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Field>
                <FieldLabel htmlFor="edit-name">Name</FieldLabel>
                <Input
                  id="edit-name"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="edit-description">Description</FieldLabel>
                <Input
                  id="edit-description"
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                />
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
                Permanently delete this organization and all its data.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ConfirmDialog
                title="Delete Organization?"
                description="Are you sure? This action cannot be undone. All data associated with this organization will be permanently deleted."
                details={[
                  { label: "Organization", value: org.name },
                  { label: "Slug", value: org.slug },
                  { label: "Members", value: String(members.length) },
                ]}
                actionLabel="Delete"
                onConfirm={handleDelete}
              >
                <Button variant="destructive">
                  <Trash2 className="mr-2 size-4" />
                  Delete Organization
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

