"use client"

import { useParams, useRouter } from "next/navigation"
import { useAuth, ROUTES } from "@/core"
import { usePublicConfigQuery } from "@/features/config"
import {
  useInvitationByToken,
  useAcceptInvitation,
  useDeclineInvitationByToken,
} from "@/features/admin"
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
  Button,
  Alert,
  AlertTitle,
  AlertDescription,
} from "@/ui"
import { CheckCircle2, XCircle, Clock, Loader2, Mail, ShieldOff } from "lucide-react"
import { toast } from "sonner"

export const InviteAcceptView = () => {
  const params = useParams<{ token: string }>()
  const router = useRouter()
  const token = params.token
  const { isAuthenticated, isLoading: authLoading, refreshWorkspaces } = useAuth()

  const { registrationEnabled, isLoading: configLoading } = usePublicConfigQuery()

  const {
    data: invitationResponse,
    isLoading: invitationLoading,
    error: invitationError,
  } = useInvitationByToken(token)

  const acceptMutation = useAcceptInvitation()
  const declineMutation = useDeclineInvitationByToken()

  const invitation = invitationResponse?.data

  const handleAccept = async () => {
    if (!invitation) return
    try {
      await acceptMutation.mutateAsync({
        workspaceId: invitation.workspaceId,
        token,
      })
      await refreshWorkspaces()
      toast.success("Invitation accepted! Redirecting to dashboard...")
      router.push(ROUTES.DASHBOARD)
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Failed to accept invitation"
      toast.error(message)
    }
  }

  const handleDecline = async () => {
    if (!invitation) return
    try {
      await declineMutation.mutateAsync({
        workspaceId: invitation.workspaceId,
        token,
      })
      toast.success("Invitation declined.")
      router.push(ROUTES.WORKSPACES)
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Failed to decline invitation"
      toast.error(message)
    }
  }

  const handleLogin = () => {
    router.push(`${ROUTES.LOGIN}?redirect=${ROUTES.INVITE}/${token}`)
  }

  const handleRegister = () => {
    router.push(`${ROUTES.REGISTER}?redirect=${ROUTES.INVITE}/${token}`)
  }

  if (authLoading || invitationLoading || configLoading) {
    return (
      <Card className="w-full max-w-md">
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  // When registration is disabled, invitations are not available
  if (!configLoading && !registrationEnabled) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <ShieldOff className="mx-auto size-12 text-muted-foreground" />
          <CardTitle className="mt-4">Invitations Not Available</CardTitle>
          <CardDescription>
            Invitations are not available. Contact your workspace admin to add you to a workspace.
          </CardDescription>
        </CardHeader>
        <CardFooter className="justify-center">
          <Button variant="outline" onClick={() => router.push(ROUTES.LOGIN)}>
            Go to Login
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (invitationError || !invitation) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <XCircle className="mx-auto size-12 text-destructive" />
          <CardTitle className="mt-4">Invitation Not Found</CardTitle>
          <CardDescription>
            This invitation link is invalid or has been revoked.
          </CardDescription>
        </CardHeader>
        <CardFooter className="justify-center">
          <Button variant="outline" onClick={() => router.push(ROUTES.LOGIN)}>
            Go to Login
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (invitation.accepted) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CheckCircle2 className="mx-auto size-12 text-green-500" />
          <CardTitle className="mt-4">Already Accepted</CardTitle>
          <CardDescription>
            This invitation has already been accepted.
          </CardDescription>
        </CardHeader>
        <CardFooter className="justify-center">
          <Button onClick={() => router.push(ROUTES.DASHBOARD)}>Go to Dashboard</Button>
        </CardFooter>
      </Card>
    )
  }

  if (invitation.expired) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <Clock className="mx-auto size-12 text-muted-foreground" />
          <CardTitle className="mt-4">Invitation Expired</CardTitle>
          <CardDescription>
            This invitation has expired. Please ask your team admin to send a
            new one.
          </CardDescription>
        </CardHeader>
        <CardFooter className="justify-center">
          <Button variant="outline" onClick={() => router.push(ROUTES.LOGIN)}>
            Go to Login
          </Button>
        </CardFooter>
      </Card>
    )
  }

  if (!isAuthenticated) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <Mail className="mx-auto size-12 text-primary" />
          <CardTitle className="mt-4">You&apos;ve Been Invited</CardTitle>
          <CardDescription>
            You&apos;ve been invited to join a workspace on Veilence-MX.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Alert>
            <AlertTitle>Invitation for</AlertTitle>
            <AlertDescription>{invitation.email}</AlertDescription>
          </Alert>
          <p className="mt-4 text-center text-sm text-muted-foreground">
            Please log in or create an account to accept this invitation.
          </p>
        </CardContent>
        <CardFooter className="flex gap-3 justify-center">
          <Button onClick={handleLogin}>Log In</Button>
          <Button variant="outline" onClick={handleRegister}>
            Create Account
          </Button>
        </CardFooter>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="text-center">
        <Mail className="mx-auto size-12 text-primary" />
        <CardTitle className="mt-4">Accept Invitation</CardTitle>
        <CardDescription>
          You&apos;ve been invited to join a workspace on Veilence-MX.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Alert>
          <AlertTitle>Invitation for</AlertTitle>
          <AlertDescription>{invitation.email}</AlertDescription>
        </Alert>
      </CardContent>
      <CardFooter className="justify-center gap-3">
        <Button
          onClick={handleAccept}
          disabled={acceptMutation.isPending || declineMutation.isPending}
        >
          {acceptMutation.isPending && (
            <Loader2 className="mr-2 size-4 animate-spin" />
          )}
          Accept Invitation
        </Button>
        <Button
          variant="outline"
          onClick={handleDecline}
          disabled={acceptMutation.isPending || declineMutation.isPending}
        >
          {declineMutation.isPending && (
            <Loader2 className="mr-2 size-4 animate-spin" />
          )}
          Decline
        </Button>
      </CardFooter>
    </Card>
  )
}
