"use client"

import { useState, useMemo } from "react"
import { useRouter } from "next/navigation"
import Image from "next/image"
import veilenceLogo from "@/../public/veilence-mx.svg"
import { useAuth, sanitizeErrorMessage } from "@/core"
import { apiChangePassword, passwordChangeSchema } from "@/domains/auth"
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
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, Eye, EyeOff, ShieldAlert } from "lucide-react"
import { ZodError } from "zod"

export const ChangePasswordForm = () => {
  const router = useRouter()
  const { user, refreshUser } = useAuth()

  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showCurrentPassword, setShowCurrentPassword] = useState(false)
  const [showNewPassword, setShowNewPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const passwordStrength = useMemo(() => {
    if (!newPassword) return null
    let score = 0
    if (newPassword.length >= 8) score++
    if (/[A-Z]/.test(newPassword)) score++
    if (/[0-9]/.test(newPassword)) score++
    if (/[^A-Za-z0-9]/.test(newPassword)) score++

    if (score <= 1) return { label: "Weak", color: "bg-red-500", width: "w-1/3" } as const
    if (score <= 2) return { label: "Medium", color: "bg-yellow-500", width: "w-2/3" } as const
    return { label: "Strong", color: "bg-green-500", width: "w-full" } as const
  }, [newPassword])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrors({})
    setServerError("")

    // Frontend same-password check
    if (currentPassword === newPassword) {
      setErrors({ newPassword: "New password must be different from your current password" })
      return
    }

    try {
      passwordChangeSchema.parse({
        currentPassword,
        newPassword,
        confirmPassword,
      })
    } catch (err) {
      if (err instanceof ZodError) {
        const fieldErrors: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") {
            fieldErrors[key] = issue.message
          }
        }
        setErrors(fieldErrors)
      }
      return
    }

    try {
      setLoading(true)
      await apiChangePassword({ currentPassword, newPassword })
      await refreshUser()
      router.push("/")
    } catch (err: unknown) {
      setServerError(sanitizeErrorMessage(err, "Failed to change password"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full max-w-sm px-4">
      <Card>
        <CardHeader className="text-center">
          <Image
            src={veilenceLogo}
            alt="Veilence-MX"
            width={40}
            height={40}
            className="mx-auto mb-2 size-10"
          />
          <CardTitle className="text-xl">Change Your Password</CardTitle>
          <CardDescription>
            {user?.mustChangePassword ? (
              <span className="flex items-center justify-center gap-1.5 text-amber-600 dark:text-amber-400">
                <ShieldAlert className="size-4" />
                You must change your password before continuing.
              </span>
            ) : (
              "Update your password to keep your account secure."
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {serverError && (
              <Alert
                variant="destructive"
                className="text-center bg-destructive/10 border-destructive"
              >
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}

            <div className="relative">
              <Field data-invalid={!!errors.currentPassword}>
                <FieldLabel htmlFor="currentPassword">
                  Current Password
                </FieldLabel>
                <Input
                  id="currentPassword"
                  type={showCurrentPassword ? "text" : "password"}
                  placeholder="Enter your current password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                />
                {errors.currentPassword && (
                  <FieldError>{errors.currentPassword}</FieldError>
                )}
              </Field>
              <Button
                variant="ghost"
                size="icon-xs"
                type="button"
                className="absolute right-2 top-7.5 text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowCurrentPassword((prev) => !prev)}
                aria-label={
                  showCurrentPassword ? "Hide password" : "Show password"
                }
                tabIndex={-1}
              >
                {showCurrentPassword ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </Button>
            </div>

            <div className="relative">
              <Field data-invalid={!!errors.newPassword}>
                <FieldLabel htmlFor="newPassword">New Password</FieldLabel>
                <Input
                  id="newPassword"
                  type={showNewPassword ? "text" : "password"}
                  placeholder="At least 8 characters"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  autoComplete="new-password"
                />
                {errors.newPassword && (
                  <FieldError>{errors.newPassword}</FieldError>
                )}
              </Field>
              <Button
                variant="ghost"
                size="icon-xs"
                type="button"
                className="absolute right-2 top-7.5 text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowNewPassword((prev) => !prev)}
                aria-label={
                  showNewPassword ? "Hide password" : "Show password"
                }
                tabIndex={-1}
              >
                {showNewPassword ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </Button>
              {passwordStrength && (
                <div className="mt-1.5 space-y-1">
                  <div className="h-1.5 w-full rounded-full bg-muted">
                    <div
                      className={`h-full rounded-full transition-all ${passwordStrength.color} ${passwordStrength.width}`}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Strength:{" "}
                    <span className="font-medium">
                      {passwordStrength.label}
                    </span>
                  </p>
                </div>
              )}
            </div>

            <div className="relative">
              <Field data-invalid={!!errors.confirmPassword}>
                <FieldLabel htmlFor="confirmPassword">
                  Confirm New Password
                </FieldLabel>
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? "text" : "password"}
                  placeholder="Confirm your new password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  autoComplete="new-password"
                />
                {errors.confirmPassword && (
                  <FieldError>{errors.confirmPassword}</FieldError>
                )}
              </Field>
              <Button
                variant="ghost"
                size="icon-xs"
                type="button"
                className="absolute right-2 top-7.5 text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowConfirmPassword((prev) => !prev)}
                aria-label={
                  showConfirmPassword ? "Hide password" : "Show password"
                }
                tabIndex={-1}
              >
                {showConfirmPassword ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </Button>
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 size-4 animate-spin" />}
              Change Password
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
