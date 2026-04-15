"use client"

import { useState } from "react"
import { useAuth } from "@/core/providers/auth-provider"
import {
  profileSchema,
  passwordChangeSchema,
} from "@/domains/auth"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/ui/components/card"
import { Button } from "@/ui/components/button"
import { Badge } from "@/ui/components/badge"
import { Separator } from "@/ui/components/separator"
import { Input } from "@/ui/components/input"
import { Field, FieldLabel, FieldError } from "@/ui/components/field"
import {
  Save,
  Loader2,
  Key,
  ShieldCheck,
  ShieldAlert,
  Mail,
  User,
  Lock,
  Monitor,
  ArrowRight,
} from "lucide-react"
import { Label } from "@/ui/components/label"
import Link from "next/link"
import {
  useUpdateProfile,
  useChangePassword,
  useSendVerification,
} from "@/features/account/hooks/use-profile"
import { useApiKeys } from "@/features/account/hooks/use-api-keys"
import { useSessions } from "@/features/account/hooks/use-sessions"
import { ZodError } from "zod"

export const AccountView = () => {

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Account</h1>
        <p className="mt-1 text-muted-foreground">
          Manage your profile, security, and API keys.
        </p>
      </div>

      {/* Profile Section */}
      <ProfileSection />

      {/* Email Verification */}
      <EmailVerificationSection />

      <Separator />

      {/* Password Change */}
      <PasswordSection />

      <Separator />

      {/* API Keys */}
      <ApiKeysSection />

      <Separator />

      {/* Active Sessions */}
      <SessionsSection />
    </div>
  )
}

// ─── Profile Section ──────────────────────────────────────────

const ProfileSection = () => {
  const { user, refreshUser } = useAuth()
  const updateProfile = useUpdateProfile()

  const [firstName, setFirstName] = useState("")
  const [lastName, setLastName] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [prevUserId, setPrevUserId] = useState<number | null>(null)

  // React-recommended "store previous props" pattern for syncing derived state
  if (user && prevUserId !== user.id) {
    setPrevUserId(user.id)
    setFirstName(user.firstName)
    setLastName(user.lastName)
  }

  const handleSave = async () => {
    setErrors({})
    try {
      const data = profileSchema.parse({ firstName, lastName })
      await updateProfile.mutateAsync(data)
      await refreshUser()
    } catch (err) {
      if (err instanceof ZodError) {
        const fieldErrors: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") fieldErrors[key] = issue.message
        }
        setErrors(fieldErrors)
      }
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <User className="size-5" />
          Profile
        </CardTitle>
        <CardDescription>
          Your personal information.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field data-invalid={!!errors.firstName}>
            <FieldLabel htmlFor="firstName">First Name</FieldLabel>
            <Input
              id="firstName"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              required
            />
            {errors.firstName && <FieldError>{errors.firstName}</FieldError>}
          </Field>
          <Field data-invalid={!!errors.lastName}>
            <FieldLabel htmlFor="lastName">Last Name</FieldLabel>
            <Input
              id="lastName"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              required
            />
            {errors.lastName && <FieldError>{errors.lastName}</FieldError>}
          </Field>
        </div>
        <div className="space-y-2">
          <Label className="text-sm font-medium">Email</Label>
          <p className="text-sm text-muted-foreground">
            {user?.email ?? "—"}
          </p>
        </div>
        <Button onClick={handleSave} disabled={updateProfile.isPending}>
          {updateProfile.isPending ? (
            <Loader2 className="mr-2 size-4 animate-spin" />
          ) : (
            <Save className="mr-2 size-4" />
          )}
          Save Profile
        </Button>
      </CardContent>
    </Card>
  )
}

// ─── Email Verification ──────────────────────────────────────

const EmailVerificationSection = () => {
  const { user } = useAuth()
  const sendVerification = useSendVerification()

  if (!user) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Mail className="size-5" />
          Email Verification
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {user.emailVerified ? (
              <>
                <ShieldCheck className="size-5 text-green-500" />
                <div>
                  <p className="font-medium text-green-700 dark:text-green-400">
                    Email verified
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {user.email}
                  </p>
                </div>
              </>
            ) : (
              <>
                <ShieldAlert className="size-5 text-orange-500" />
                <div>
                  <p className="font-medium text-orange-700 dark:text-orange-400">
                    Email not verified
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {user.email} — please verify your email address
                  </p>
                </div>
              </>
            )}
          </div>
          {!user.emailVerified && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => sendVerification.mutate()}
              disabled={sendVerification.isPending}
            >
              {sendVerification.isPending && (
                <Loader2 className="mr-2 size-4 animate-spin" />
              )}
              Send Verification
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// ─── Password Section ──────────────────────────────────────────

const PasswordSection = () => {
  const changePassword = useChangePassword()
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrors({})
    try {
      const data = passwordChangeSchema.parse({
        currentPassword,
        newPassword,
        confirmPassword,
      })
      await changePassword.mutateAsync({
        currentPassword: data.currentPassword,
        newPassword: data.newPassword,
      })
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch (err) {
      if (err instanceof ZodError) {
        const fieldErrors: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") fieldErrors[key] = issue.message
        }
        setErrors(fieldErrors)
      }
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Lock className="size-5" />
          Change Password
        </CardTitle>
        <CardDescription>
          Update your password to keep your account secure.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="max-w-md space-y-4">
          <Field data-invalid={!!errors.currentPassword}>
            <FieldLabel htmlFor="currentPassword">Current Password</FieldLabel>
            <Input
              id="currentPassword"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
            {errors.currentPassword && <FieldError>{errors.currentPassword}</FieldError>}
          </Field>
          <Field data-invalid={!!errors.newPassword}>
            <FieldLabel htmlFor="newPassword">New Password</FieldLabel>
            <Input
              id="newPassword"
              type="password"
              placeholder="At least 8 characters"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
            {errors.newPassword && <FieldError>{errors.newPassword}</FieldError>}
          </Field>
          <Field data-invalid={!!errors.confirmPassword}>
            <FieldLabel htmlFor="confirmPassword">Confirm New Password</FieldLabel>
            <Input
              id="confirmPassword"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              autoComplete="new-password"
            />
            {errors.confirmPassword && <FieldError>{errors.confirmPassword}</FieldError>}
          </Field>
          <Button type="submit" disabled={changePassword.isPending}>
            {changePassword.isPending && (
              <Loader2 className="mr-2 size-4 animate-spin" />
            )}
            Update Password
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

// ─── API Keys Summary ──────────────────────────────────────────

const ApiKeysSection = () => {
  const { data: keysRes, isLoading } = useApiKeys()

  const keys = keysRes?.data ?? []
  const activeCount = keys.filter((k) => k.isActive).length

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Key className="size-5" />
          API Keys
        </CardTitle>
        <CardDescription>
          Manage API keys for programmatic access.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isLoading ? (
              <Loader2 className="size-4 animate-spin text-muted-foreground" />
            ) : (
              <>
                <Badge variant="secondary" className="text-sm">
                  {activeCount} active {activeCount === 1 ? "key" : "keys"}
                </Badge>
                {keys.length > activeCount && (
                  <span className="text-xs text-muted-foreground">
                    ({keys.length} total)
                  </span>
                )}
              </>
            )}
          </div>
          <Link href="/settings/api-keys">
            <Button variant="outline" size="sm">
              Manage API Keys
              <ArrowRight className="ml-2 size-4" />
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

// ─── Sessions Summary ──────────────────────────────────────────

const SessionsSection = () => {
  const { data: sessionsRes, isLoading } = useSessions()

  const sessions = sessionsRes?.data ?? []
  const activeCount = sessions.length

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Monitor className="size-5" />
          Active Sessions
        </CardTitle>
        <CardDescription>
          View and manage your active sessions across devices.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isLoading ? (
              <Loader2 className="size-4 animate-spin text-muted-foreground" />
            ) : (
              <Badge variant="secondary" className="text-sm">
                {activeCount} active {activeCount === 1 ? "session" : "sessions"}
              </Badge>
            )}
          </div>
          <Link href="/settings/sessions">
            <Button variant="outline" size="sm">
              Manage Sessions
              <ArrowRight className="ml-2 size-4" />
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}
