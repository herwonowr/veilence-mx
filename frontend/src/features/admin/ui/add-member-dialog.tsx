"use client"

import { useState } from "react"
import { sanitizeErrorMessage } from "@/core"
import type { Role } from "@/domains/admin"
import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldError,
  Alert,
  AlertDescription,
  Checkbox,
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
import { UserPlus, Loader2, Eye, EyeOff } from "lucide-react"
import { useAddMember } from "@/features/admin"

interface AddMemberDialogProps {
  workspaceId: string
  roles: Role[]
  capitalize: (s: string) => string
}

export const AddMemberDialog = ({
  workspaceId,
  roles,
  capitalize,
}: AddMemberDialogProps) => {
  const [open, setOpen] = useState(false)
  const [email, setEmail] = useState("")
  const [firstName, setFirstName] = useState("")
  const [lastName, setLastName] = useState("")
  const [roleId, setRoleId] = useState<string | null>(null)
  const [setPassword, setSetPassword] = useState(false)
  const [password, setPassword_] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState("")
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const addMemberMutation = useAddMember()

  const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError("")
    setFieldErrors({})

    const errs: Record<string, string> = {}
    if (!email.trim()) errs.email = "Email is required"
    if (!firstName.trim()) errs.firstName = "First name is required"
    if (!lastName.trim()) errs.lastName = "Last name is required"
    if (!roleId) errs.roleId = "Please select a role"
    if (setPassword && password.length < 8) {
      errs.password = "Password must be at least 8 characters"
    }
    if (setPassword && password.length > 72) {
      errs.password = "Password must be at most 72 characters"
    }
    if (Object.keys(errs).length > 0) {
      setFieldErrors(errs)
      return
    }

    try {
      await addMemberMutation.mutateAsync({
        workspaceId,
        data: {
          email,
          firstName,
          lastName,
          roleId: roleId!,
          ...(setPassword && password ? { password } : {}),
        },
      })
      setOpen(false)
      setEmail("")
      setFirstName("")
      setLastName("")
      setRoleId(null)
      setSetPassword(false)
      setPassword_("")
    } catch (err) {
      setError(sanitizeErrorMessage(err, "Failed to add member"))
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        setOpen(isOpen)
        if (!isOpen) {
          setPassword_("")
          setShowPassword(false)
        }
      }}
    >
      <DialogTrigger
        render={
          <Button size="sm">
            <UserPlus className="mr-2 size-4" />
            Add Member
          </Button>
        }
      />
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Member</DialogTitle>
            <DialogDescription>
              Create a user account and add them to this workspace.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            {error && (
              <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <Field data-invalid={!!fieldErrors.email}>
              <FieldLabel htmlFor="add-member-email">Email</FieldLabel>
              <Input
                id="add-member-email"
                type="email"
                placeholder="user@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
              {fieldErrors.email && <FieldError>{fieldErrors.email}</FieldError>}
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field data-invalid={!!fieldErrors.firstName}>
                <FieldLabel htmlFor="add-member-first-name">First Name</FieldLabel>
                <Input
                  id="add-member-first-name"
                  placeholder="First name"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                />
                {fieldErrors.firstName && <FieldError>{fieldErrors.firstName}</FieldError>}
              </Field>
              <Field data-invalid={!!fieldErrors.lastName}>
                <FieldLabel htmlFor="add-member-last-name">Last Name</FieldLabel>
                <Input
                  id="add-member-last-name"
                  placeholder="Last name"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                />
                {fieldErrors.lastName && <FieldError>{fieldErrors.lastName}</FieldError>}
              </Field>
            </div>
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
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Checkbox
                  id="add-member-set-password"
                  checked={setPassword}
                  onCheckedChange={(checked) => {
                    setSetPassword(checked === true)
                    if (!checked) {
                      setPassword_("")
                      setShowPassword(false)
                    }
                  }}
                />
                <label
                  htmlFor="add-member-set-password"
                  className="text-sm font-medium leading-none cursor-pointer"
                >
                  Set initial password
                </label>
              </div>
              {setPassword && (
                <div className="relative">
                  <Field data-invalid={!!fieldErrors.password}>
                    <FieldLabel htmlFor="add-member-password">Password</FieldLabel>
                    <Input
                      id="add-member-password"
                      type={showPassword ? "text" : "password"}
                      placeholder="At least 8 characters"
                      value={password}
                      onChange={(e) => setPassword_(e.target.value)}
                    />
                    {fieldErrors.password && <FieldError>{fieldErrors.password}</FieldError>}
                  </Field>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    type="button"
                    className="absolute right-2 top-7.5 text-muted-foreground hover:text-foreground transition-colors"
                    onClick={() => setShowPassword((prev) => !prev)}
                    aria-label={showPassword ? "Hide password" : "Show password"}
                    tabIndex={-1}
                  >
                    {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </Button>
                  <p className="text-xs text-muted-foreground mt-1">
                    User will be required to change this password on first login.
                  </p>
                </div>
              )}
              {!setPassword && (
                <p className="text-xs text-muted-foreground">
                  User will receive an email to set their own password.
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={addMemberMutation.isPending || !roleId}>
              {addMemberMutation.isPending && (
                <Loader2 className="mr-2 size-4 animate-spin" />
              )}
              Add Member
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
