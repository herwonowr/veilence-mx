"use client"

import { useState } from "react"
import { sanitizeErrorMessage } from "@/core"
import { invitationSchema } from "@/domains/admin"
import type { Role } from "@/domains/admin"
import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldError,
  Alert,
  AlertDescription,
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import { UserPlus, Loader2 } from "lucide-react"
import { ZodError } from "zod"
import { useInviteMember } from "@/features/admin"

interface InviteMemberDialogProps {
  workspaceId: string
  roles: Role[]
  capitalize: (s: string) => string
}

export const InviteMemberDialog = ({
  workspaceId,
  roles,
  capitalize,
}: InviteMemberDialogProps) => {
  const [open, setOpen] = useState(false)
  const [email, setEmail] = useState("")
  const [roleId, setRoleId] = useState<string | null>(null)
  const [error, setError] = useState("")
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const inviteMutation = useInviteMember()

  const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError("")
    setFieldErrors({})
    try {
      invitationSchema.parse({ email, roleId: roleId ? Number(roleId) : undefined })
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
    if (!roleId) return
    try {
      await inviteMutation.mutateAsync({
        workspaceId,
        data: { email, roleId },
      })
      setOpen(false)
      setEmail("")
      setRoleId(null)
    } catch (err) {
      setError(sanitizeErrorMessage(err, "Failed to send invitation"))
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => setOpen(isOpen)}
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
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Invite Member</DialogTitle>
            <DialogDescription>
              Send an invitation to join this workspace.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            {error && (
              <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <Field data-invalid={!!fieldErrors.email}>
              <FieldLabel htmlFor="invite-email">Email</FieldLabel>
              <Input
                id="invite-email"
                type="email"
                placeholder="user@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
              {fieldErrors.email && <FieldError>{fieldErrors.email}</FieldError>}
            </Field>
            <Field data-invalid={!!fieldErrors.roleId}>
              <FieldLabel>Role</FieldLabel>
              <Select
                value={roleId != null ? String(roleId) : undefined}
                onValueChange={(v) => setRoleId(v || null)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue>{roleId != null ? capitalize(roles.find(r => r.id === roleId)?.name ?? "") : "Select a role"}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {roles.filter((role) => role.name.toLowerCase() !== "owner").map((role) => (
                    <SelectItem key={role.id} value={String(role.id)}>
                      {capitalize(role.name)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {fieldErrors.roleId && <FieldError>{fieldErrors.roleId}</FieldError>}
            </Field>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={inviteMutation.isPending || !roleId}>
              {inviteMutation.isPending && (
                <Loader2 className="mr-2 size-4 animate-spin" />
              )}
              Send Invitation
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
