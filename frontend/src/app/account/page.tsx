"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import {
  profileSchema,
  passwordChangeSchema,
} from "@/lib/validations"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { FormField } from "@/components/form-field"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Save,
  Loader2,
  Key,
  Plus,
  Trash2,
  ShieldCheck,
  ShieldAlert,
  Mail,
  User,
  Lock,
} from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import {
  useUpdateProfile,
  useChangePassword,
  useSendVerification,
  useApiKeys,
  useCreateApiKey,
  useDeleteApiKey,
} from "@/features/account"
import { ZodError } from "zod"

export default function AccountPage() {
  return (
    <ProtectedRoute>
      <AccountContent />
    </ProtectedRoute>
  )
}

function AccountContent() {
  const { user, refreshUser } = useAuth()

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
    </div>
  )
}

// ─── Profile Section ──────────────────────────────────────────

function ProfileSection() {
  const { user, refreshUser } = useAuth()
  const updateProfile = useUpdateProfile()

  const [firstName, setFirstName] = useState("")
  const [lastName, setLastName] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [initialized, setInitialized] = useState(false)

  useEffect(() => {
    if (user && !initialized) {
      setFirstName(user.firstName)
      setLastName(user.lastName)
      setInitialized(true)
    }
  }, [user, initialized])

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
          <FormField
            id="firstName"
            label="First Name"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
            error={errors.firstName}
            required
          />
          <FormField
            id="lastName"
            label="Last Name"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
            error={errors.lastName}
            required
          />
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

function EmailVerificationSection() {
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

function PasswordSection() {
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
          <FormField
            id="currentPassword"
            label="Current Password"
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            error={errors.currentPassword}
            required
            autoComplete="current-password"
          />
          <FormField
            id="newPassword"
            label="New Password"
            type="password"
            placeholder="At least 8 characters"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            error={errors.newPassword}
            required
            autoComplete="new-password"
          />
          <FormField
            id="confirmPassword"
            label="Confirm New Password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={errors.confirmPassword}
            required
            autoComplete="new-password"
          />
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

// ─── API Keys Section ──────────────────────────────────────────

function ApiKeysSection() {
  const { data: keysRes, isLoading } = useApiKeys()
  const createMutation = useCreateApiKey()
  const deleteMutation = useDeleteApiKey()

  const [createOpen, setCreateOpen] = useState(false)
  const [keyName, setKeyName] = useState("")
  const [newKeyValue, setNewKeyValue] = useState<string | null>(null)

  const keys = keysRes?.data ?? []

  const handleCreate = async () => {
    if (!keyName) return
    try {
      const res = await createMutation.mutateAsync({ name: keyName })
      setNewKeyValue(res.data.apiKey)
      setKeyName("")
      setCreateOpen(false)
    } catch {
      // Error handled by mutation
    }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="flex items-center gap-2">
            <Key className="size-5" />
            API Keys
          </CardTitle>
          <CardDescription>
            Manage API keys for programmatic access.
          </CardDescription>
        </div>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger render={<Button size="sm" />}>
            <Plus className="mr-1 size-4" />
            Create Key
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create API Key</DialogTitle>
              <DialogDescription>
                Give your key a descriptive name.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <Label htmlFor="key-name">Key Name</Label>
                <Input
                  id="key-name"
                  placeholder="e.g., CI/CD Pipeline"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                onClick={handleCreate}
                disabled={!keyName || createMutation.isPending}
              >
                {createMutation.isPending && (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                )}
                Create
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent>
        {newKeyValue && (
          <div className="mb-4 rounded-md border border-green-200 bg-green-50 p-3 dark:border-green-900 dark:bg-green-950/30">
            <p className="mb-1 text-sm font-medium text-green-800 dark:text-green-300">
              API key created — copy it now, it won&apos;t be shown again:
            </p>
            <code className="block break-all rounded bg-green-100 px-2 py-1 text-xs dark:bg-green-900/50">
              {newKeyValue}
            </code>
            <Button
              variant="ghost"
              size="sm"
              className="mt-2"
              onClick={() => setNewKeyValue(null)}
            >
              Dismiss
            </Button>
          </div>
        )}
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="size-6 animate-spin text-muted-foreground" />
          </div>
        ) : keys.length === 0 ? (
          <p className="py-8 text-center text-muted-foreground">
            No API keys. Create one for programmatic access.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Key</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Last Used</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-16" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell className="font-medium">{key.name}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {key.keyPrefix}...
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(key.createdAt).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {key.lastUsedAt
                      ? new Date(key.lastUsedAt).toLocaleDateString()
                      : "Never"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={key.isActive ? "default" : "secondary"}>
                      {key.isActive ? "Active" : "Revoked"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {key.isActive && (
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => deleteMutation.mutate(key.id)}
                        disabled={deleteMutation.isPending}
                      >
                        <Trash2 className="size-4 text-destructive" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
