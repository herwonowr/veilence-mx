"use client"

import { useEffect, useMemo, useState } from "react"
import { useRouter } from "next/navigation"
import Image from "next/image"
import veilenceLogo from "@/../public/veilence-mx.svg"
import Link from "next/link"
import { useAuth, ROUTES , usePublicConfigQuery, parseFieldErrors } from "@/core"
import { registerSchema, getPasswordStrength } from "@/domains/auth"
import { Button, Input, Field, FieldLabel, FieldError, Alert, AlertDescription } from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, Eye, EyeOff, MailCheck, Info } from "lucide-react"

export const RegisterForm = () => {
  const { register } = useAuth()
  const router = useRouter()

  const { setupRequired, registrationEnabled, hasEmailDomainRestriction, isLoading: configLoading } = usePublicConfigQuery()

  useEffect(() => {
    if (configLoading) return
    if (setupRequired) {
      router.replace(ROUTES.SETUP)
      return
    }
    if (!registrationEnabled) {
      router.replace(ROUTES.LOGIN)
    }
  }, [configLoading, setupRequired, registrationEnabled, router])

  const [firstName, setFirstName] = useState("")
  const [lastName, setLastName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formSubmitted, setFormSubmitted] = useState(false)
  const [serverError, setServerError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [registrationSuccess, setRegistrationSuccess] = useState(false)

  const passwordStrength = useMemo(() => getPasswordStrength(password), [password])

  const validate = (fields: { firstName: string; lastName: string; email: string; password: string; confirmPassword: string }) => {
    if (!formSubmitted) return
    const result = registerSchema.safeParse(fields)
    setErrors(result.success ? {} : parseFieldErrors(result.error))
  }

  const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setFormSubmitted(true)
    setErrors({})
    setServerError("")

    try {
      const data = registerSchema.parse({
        firstName,
        lastName,
        email,
        password,
        confirmPassword,
      })
      setLoading(true)
      const result = await register({
        email: data.email,
        password: data.password,
        firstName: data.firstName,
        lastName: data.lastName,
      })
      // If the backend created the account but could not generate tokens,
      // it returns a code instructing the user to log in manually.
      if (result?.code === "registration_complete_login_required" || result?.code === "email_verification_required") {
        try { sessionStorage.setItem("vmx_just_registered", "true") } catch {}
        setFormSubmitted(false)
        setRegistrationSuccess(true)
        return
      }
      router.push(`${ROUTES.WORKSPACES}?create=true`)
    } catch (err) {
      const fieldErrors = parseFieldErrors(err)
      if (Object.keys(fieldErrors).length > 0) {
        setErrors(fieldErrors)
      } else {
        setServerError(
          err instanceof Error ? err.message : "Registration failed"
        )
      }
    } finally {
      setLoading(false)
    }
  }

  // Don't render while checking config or if registration is disabled
  if (configLoading || !registrationEnabled) {
    return null
  }

  if (registrationSuccess) {
    return (
      <div className="w-full max-w-sm px-4">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-green-600 text-white">
              <MailCheck className="size-5" />
            </div>
            <CardTitle className="text-xl">Registration successful!</CardTitle>
            <CardDescription>
              Check your email for a verification link. You need to verify your
              email address before you can sign in.
            </CardDescription>
          </CardHeader>
          <CardContent className="text-center">
            <Link href={ROUTES.LOGIN}>
              <Button variant="outline" className="w-full">
                Go to Sign In
              </Button>
            </Link>
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
          <CardTitle className="text-xl">Create an account</CardTitle>
          <CardDescription>
            Get started with Veilence-MX
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} noValidate className="space-y-4">
            {hasEmailDomainRestriction && (
              <Alert variant="default" className="bg-blue-50 border-blue-200 dark:bg-blue-950/30 dark:border-blue-800">
                <Info className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                <AlertDescription className="text-blue-800 dark:text-blue-300">
                  Email domain restrictions apply. Contact your admin if you have issues.
                </AlertDescription>
              </Alert>
            )}
            {serverError && (
              <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}
            <Field data-invalid={!!errors.firstName}>
              <FieldLabel htmlFor="firstName">First Name</FieldLabel>
              <Input
                id="firstName"
                placeholder="First name"
                value={firstName}
                onChange={(e) => { setFirstName(e.target.value); validate({ firstName: e.target.value, lastName, email, password, confirmPassword }) }}
                autoComplete="given-name"
              />
              {errors.firstName && <FieldError>{errors.firstName}</FieldError>}
            </Field>
            <Field data-invalid={!!errors.lastName}>
              <FieldLabel htmlFor="lastName">Last Name</FieldLabel>
              <Input
                id="lastName"
                placeholder="Last name"
                value={lastName}
                onChange={(e) => { setLastName(e.target.value); validate({ firstName, lastName: e.target.value, email, password, confirmPassword }) }}
                autoComplete="family-name"
              />
              {errors.lastName && <FieldError>{errors.lastName}</FieldError>}
            </Field>
            <Field data-invalid={!!errors.email}>
              <FieldLabel htmlFor="email">Email</FieldLabel>
              <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => { setEmail(e.target.value); validate({ firstName, lastName, email: e.target.value, password, confirmPassword }) }}
                autoComplete="email"
              />
              {errors.email && <FieldError>{errors.email}</FieldError>}
            </Field>
            <div className="relative">
              <Field data-invalid={!!errors.password}>
                <FieldLabel htmlFor="password">Password</FieldLabel>
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder="At least 8 characters"
                  value={password}
                  onChange={(e) => { setPassword(e.target.value); validate({ firstName, lastName, email, password: e.target.value, confirmPassword }) }}
                  autoComplete="new-password"
                />
                {errors.password && <FieldError>{errors.password}</FieldError>}
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
                {showPassword ? (
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
                    Strength: <span className="font-medium">{passwordStrength.label}</span>
                  </p>
                </div>
              )}
            </div>
            <div className="relative">
              <Field data-invalid={!!errors.confirmPassword}>
                <FieldLabel htmlFor="confirmPassword">Confirm Password</FieldLabel>
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? "text" : "password"}
                  placeholder="Confirm your password"
                  value={confirmPassword}
                  onChange={(e) => { setConfirmPassword(e.target.value); validate({ firstName, lastName, email, password, confirmPassword: e.target.value }) }}
                  autoComplete="new-password"
                />
                {errors.confirmPassword && <FieldError>{errors.confirmPassword}</FieldError>}
              </Field>
              <Button
                variant="ghost"
                size="icon-xs"
                type="button"
                className="absolute right-2 top-7.5 text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowConfirmPassword((prev) => !prev)}
                aria-label={showConfirmPassword ? "Hide password" : "Show password"}
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
              Create Account
            </Button>
          </form>
          <div className="mt-4 text-center text-sm text-muted-foreground">
            Already have an account?{" "}
            <Link
              href={ROUTES.LOGIN}
              className="font-medium text-primary underline-offset-4 hover:underline"
            >
              Sign in
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
