"use client"

import { useRouter } from "next/navigation"
import { useAuth, ROUTES } from "@/core"
import { usePublicConfigQuery } from "@/features/config"
import type { MyInvitation } from "@/domains/admin"
import {
  useMyInvitations,
  useAcceptInvitationById,
  useDeclineInvitationById,
} from "@/features/admin"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  Button,
  Badge,
  EmptyState,
} from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import { Mail, Loader2, Check, X } from "lucide-react"

export const MyInvitationsView = () => {
  const router = useRouter()
  const { refreshWorkspaces } = useAuth()
  const { data: invitationsRes, isLoading } = useMyInvitations()
  const acceptMutation = useAcceptInvitationById()
  const declineMutation = useDeclineInvitationById()

  const { registrationEnabled } = usePublicConfigQuery()

  const invitations = invitationsRes?.data ?? []
  const pendingInvitations = invitations.filter((inv) => inv.status === "pending")

  // When registration is disabled, show empty state
  if (!registrationEnabled) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">My Invitations</h1>
            <p className="text-muted-foreground mt-1">
              Invitations are not available when registration is disabled.
            </p>
          </div>
          <Button variant="outline" onClick={() => router.push(ROUTES.WORKSPACES)}>
            Back to Workspaces
          </Button>
        </div>
        <Card>
          <CardContent>
            <EmptyState
              icon={<Mail className="h-12 w-12" />}
              title="Invitations not available"
              description="Invitations are not available when registration is disabled. Contact your workspace admin to be added directly."
            >
              <Button variant="outline" onClick={() => router.push(ROUTES.WORKSPACES)}>
                Go to Workspaces
              </Button>
            </EmptyState>
          </CardContent>
        </Card>
      </div>
    )
  }

  const handleAccept = async (invitation: MyInvitation) => {
    await acceptMutation.mutateAsync(invitation.id)
    await refreshWorkspaces()
    router.push(ROUTES.WORKSPACES)
  }

  const handleDecline = async (invitation: MyInvitation) => {
    await declineMutation.mutateAsync(invitation.id)
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">My Invitations</h1>
          <p className="text-muted-foreground mt-1">
            Pending workspace invitations sent to you.
          </p>
        </div>
        <Button variant="outline" onClick={() => router.push(ROUTES.WORKSPACES)}>
          Back to Workspaces
        </Button>
      </div>

      {pendingInvitations.length === 0 ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<Mail className="h-12 w-12" />}
              title="No pending invitations"
              description="You don't have any pending workspace invitations."
            >
              <Button variant="outline" onClick={() => router.push(ROUTES.WORKSPACES)}>
                Go to Workspaces
              </Button>
            </EmptyState>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Pending Invitations</CardTitle>
            <CardDescription>
              {pendingInvitations.length} pending{" "}
              {pendingInvitations.length === 1 ? "invitation" : "invitations"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Workspace</TableHead>
                  <TableHead>Invited By</TableHead>
                  <TableHead>Invited Date</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pendingInvitations.map((invitation) => (
                  <InvitationRow
                    key={invitation.id}
                    invitation={invitation}
                    onAccept={handleAccept}
                    onDecline={handleDecline}
                    isAccepting={acceptMutation.isPending}
                    isDeclining={declineMutation.isPending}
                  />
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

const InvitationRow = ({
  invitation,
  onAccept,
  onDecline,
  isAccepting,
  isDeclining,
}: {
  invitation: MyInvitation
  onAccept: (inv: MyInvitation) => void
  onDecline: (inv: MyInvitation) => void
  isAccepting: boolean
  isDeclining: boolean
}) => {
  const isExpired = new Date(invitation.expiresAt) < new Date()

  return (
    <TableRow>
      <TableCell className="font-medium">{invitation.workspaceName}</TableCell>
      <TableCell>{invitation.invitedByEmail}</TableCell>
      <TableCell>{new Date(invitation.createdAt).toLocaleString()}</TableCell>
      <TableCell>
        {isExpired ? (
          <Badge variant="destructive">Expired</Badge>
        ) : (
          new Date(invitation.expiresAt).toLocaleString()
        )}
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-2">
          <Button
            size="sm"
            onClick={() => onAccept(invitation)}
            disabled={isAccepting || isDeclining || isExpired}
          >
            {isAccepting ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <Check className="mr-1 size-3" />
            )}
            Accept
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => onDecline(invitation)}
            disabled={isAccepting || isDeclining}
          >
            {isDeclining ? (
              <Loader2 className="mr-1 size-3 animate-spin" />
            ) : (
              <X className="mr-1 size-3" />
            )}
            Decline
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
}
