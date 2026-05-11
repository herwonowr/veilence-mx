"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import Image from "next/image"
import veilenceLogo from "@/../public/veilence-mx.svg"
import Link from "next/link"
import { apiForgotPassword, passwordResetSchema } from "@/domains/auth"
import { sanitizeErrorMessage, ROUTES , usePublicConfigQuery, parseFieldErrors } from "@/core"
import { Button, Input, Field, FieldLabel, FieldError, Alert, AlertDescription } from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, ArrowLeft, CheckCircle2 } from "lucide-react"

export const ForgotPasswordForm = () => {
  const router = useRouter()

  const { setupRequired } = usePublicConfigQuery()

  useEffect(() => {
    if (setupRequired) {
      router.replace(ROUTES.SETUP)
    }
  }, [setupRequired, router])

  const [email, setEmail] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState("")
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [formSubmitted, setFormSubmitted] = useState(false)

  const validate = (fields: { email: string }) => {
    if (!formSubmitted) return
    const result = passwordResetSchema.safeParse(fields)
    setErrors(result.success ? {} : parseFieldErrors(result.error))
  }

  const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setFormSubmitted(true)
    setErrors({})
    setServerError("")

    try {
      const data = passwordResetSchema.parse({ email })
      setLoading(true)
      await apiForgotPassword(data.email)
      setFormSubmitted(false)
      setSubmitted(true)
    } catch (err) {
      const fieldErrors = parseFieldErrors(err)
      if (Object.keys(fieldErrors).length > 0) {
        setErrors(fieldErrors)
      } else {
        setServerError(sanitizeErrorMessage(err, "Failed to send reset email"))
      }
    } finally {
      setLoading(false)
    }
  }

  if (submitted) {
    return (
      <div className="w-full max-w-sm px-4">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400">
              <CheckCircle2 className="size-5" />
            </div>
            <CardTitle className="text-xl">Check your email</CardTitle>
            <CardDescription>
              If an account exists for {email}, we&apos;ve sent a password reset
              link. Check your inbox and spam folder.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <Button
                variant="outline"
                className="w-full"
                onClick={() => {
                  setSubmitted(false)
                  setFormSubmitted(false)
                  setEmail("")
                }}
              >
                Send another link
              </Button>
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

  return (
    <div className="w-full max-w-sm px-4">
      <Card>
        <CardHeader className="text-center">
          <Image src={veilenceLogo} alt="Veilence-MX" width={40} height={40} className="mx-auto mb-2 size-10" />
          <CardTitle className="text-xl">Reset your password</CardTitle>
          <CardDescription>
            Enter your email address and we&apos;ll send you a link to reset
            your password.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} noValidate className="space-y-4">
            {serverError && (
              <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}
            <Field data-invalid={!!errors.email}>
              <FieldLabel htmlFor="email">Email</FieldLabel>
              <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => { setEmail(e.target.value); validate({ email: e.target.value }) }}
                autoComplete="email"
              />
              {errors.email && <FieldError>{errors.email}</FieldError>}
            </Field>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading && <Loader2 className="mr-2 size-4 animate-spin" />}
              Send Reset Link
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
