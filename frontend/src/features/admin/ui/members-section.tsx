"use client"

import { useState } from "react"
import type { WorkspaceMember, Role, Invitation, Workspace } from "@/domains/admin"
import {
  Button,
  Badge,
  ConfirmDialog,
  TableEmptyState,
  type ConfirmDialogDetail,
} from "@/ui"
import {
  Card,
  CardContent,
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
import { UserMinus, Loader2, Mail } from "lucide-react"
import {
  useRemoveMember,
  useUpdateMemberRole,
  useRevokeInvitation,
  useResendInvitation,
} from "@/features/admin"

interface MembersSectionProps {
  workspaceId: string
  workspace: Workspace
  members: WorkspaceMember[]
  roles: Role[]
  invitations: Invitation[]
  currentUserId: string | undefined
  canInvite: boolean
  canRemove: boolean
  canUpdateRole: boolean
  canManageMembers: boolean
  registrationEnabled: boolean
  capitalize: (s: string) => string
}

export const MembersSection = ({
  workspaceId,
  workspace,
  members,
  roles,
  invitations,
  currentUserId,
  canInvite,
  canRemove,
  canUpdateRole,
  canManageMembers,
  registrationEnabled,
  capitalize,
}: MembersSectionProps) => {
  const [memberToRemove, setMemberToRemove] = useState<{
    userId: string
    name: string
    details: ConfirmDialogDetail[]
  } | null>(null)

  const removeMutation = useRemoveMember()
  const updateRoleMutation = useUpdateMemberRole()
  const revokeMutation = useRevokeInvitation()
  const resendMutation = useResendInvitation()

  const confirmRemoveMember = (userId: string, firstName?: string, lastName?: string, email?: string) => {
    const name = [firstName, lastName].filter(Boolean).join(" ") || "this member"
    setMemberToRemove({
      userId,
      name,
      details: [
        { label: "Member", value: name },
        ...(email ? [{ label: "Email", value: email }] : []),
        { label: "Workspace", value: workspace.name },
      ],
    })
  }

  const handleRemoveMember = async (userId: string) => {
    await removeMutation.mutateAsync({ workspaceId, userId })
    setMemberToRemove(null)
  }

  const handleUpdateRole = (userId: string, roleId: string) => {
    updateRoleMutation.mutate({ workspaceId, userId, roleId })
  }

  return (
    <>
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
                        disabled={member.userId === currentUserId}
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
                    {member.userId !== currentUserId && canRemove && (
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

      {/* Invitations - only visible when registration is enabled */}
      {registrationEnabled && (
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
                                  workspaceId,
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
                                  workspaceId,
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
      )}

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
    </>
  )
}
