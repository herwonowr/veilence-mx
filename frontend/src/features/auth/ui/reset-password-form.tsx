"use client"

import { Suspense, useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { ROUTES , usePublicConfigQuery, parseFieldErrors } from "@/core"
import Image from "next/image"
import veilenceLogo from "@/../public/veilence-mx.svg"
import Link from "next/link"
import { apiResetPassword, newPasswordSchema } from "@/domains/auth"
import { Button, Input, Field, FieldLabel, FieldError, Alert, AlertDescription } from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, ArrowLeft, AlertTriangle } from "lucide-react"
import { toast } from "sonner"

const ResetPasswordFormInner = () => {
  const router = useRouter()
  const searchParams = useSearchParams()
  const token = searchParams.get("token")

  const { setupRequired } = usePublicConfigQuery()

  useEffect(() => {
    if (setupRequired) {
      router.replace(ROUTES.SETUP)
    }
  }, [setupRequired, router])

  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formSubmitted, setFormSubmitted] = useState(false)
  const [serverError, setServerError] = useState("")
  const [tokenInvalid, setTokenInvalid] = useState(!token)
  const [loading, setLoading] = useState(false)

  const validate = (fields: { password: string; confirmPassword: string }) => {
    if (!formSubmitted) return
    const result = newPasswordSchema.safeParse(fields)
    setErrors(result.success ? {} : parseFieldErrors(result.error))
  }

  if (tokenInvalid) {
    return (
      <div className="w-full max-w-sm px-4">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-destructive/10 text-destructive">
              <AlertTriangle className="size-5" />
            </div>
            <CardTitle className="text-xl">Invalid Reset Link</CardTitle>
            <CardDescription>
              This password reset link is invalid or has expired. Please request
              a new one.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <Link href={ROUTES.FORGOT_PASSWORD} className="block">
                <Button className="w-full">Request New Link</Button>
              </Link>
              <div className="text-center">
                <Link
                  href={ROUTES.LOGIN}
                  className="w-fit inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-primary"
                >
                  <ArrowLeft className="size-3" />
                  Back to sign in
                </Link>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setFormSubmitted(true)
    setErrors({})
    setServerError("")

    try {
      const data = newPasswordSchema.parse({ password, confirmPassword })
      setLoading(true)
      await apiResetPassword(token!, data.password)
      toast.success("Password reset successfully. You can now sign in.")
      router.push(ROUTES.LOGIN)
    } catch (err) {
      const fieldErrors = parseFieldErrors(err)
      if (Object.keys(fieldErrors).length > 0) {
        setErrors(fieldErrors)
      } else {
        const message = err instanceof Error
          ? err.message
          : "Failed to reset password. The link may have expired."
        if (/reset link|expired|invalid.*token/i.test(message)) {
          setTokenInvalid(true)
        } else {
          setServerError(message)
        }
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full max-w-sm px-4">
      <Card>
        <CardHeader className="text-center">
          <Image src={veilenceLogo} alt="Veilence-MX" width={40} height={40} className="mx-auto mb-2 size-10" />
          <CardTitle className="text-xl">Set new password</CardTitle>
          <CardDescription>
            Enter your new password below.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} noValidate className="space-y-4">
            {serverError && (
              <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}
            <Field data-invalid={!!errors.password}>
              <FieldLabel htmlFor="password">New Password</FieldLabel>
              <Input
                id="password"
                type="password"
                placeholder="At least 8 characters"
                value={password}
                onChange={(e) => { setPassword(e.target.value); validate({ password: e.target.value, confirmPassword }) }}
                autoComplete="new-password"
              />
              {errors.password && <FieldError>{errors.password}</FieldError>}
            </Field>
            <Field data-invalid={!!errors.confirmPassword}>
              <FieldLabel htmlFor="confirmPassword">Confirm New Password</FieldLabel>
              <Input
                id="confirmPassword"
                type="password"
                placeholder="Confirm new password"
                value={confirmPassword}
                onChange={(e) => { setConfirmPassword(e.target.value); validate({ password, confirmPassword: e.target.value }) }}
                autoComplete="new-password"
              />
              {errors.confirmPassword && <FieldError>{errors.confirmPassword}</FieldError>}
            </Field>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 size-4 animate-spin" />}
              Reset Password
            </Button>
          </form>
          <div className="mt-4 text-center">
            <Link
              href={ROUTES.LOGIN}
              className="w-fit inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-primary"
            >
              <ArrowLeft className="size-3" />
              Back to sign in
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export const ResetPasswordForm = () => (
  <Suspense>
    <ResetPasswordFormInner />
  </Suspense>
)
