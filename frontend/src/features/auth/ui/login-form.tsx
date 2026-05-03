"use client"

import { Suspense, useCallback, useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Image from "next/image"
import veilenceLogo from "@/../public/veilence-mx.svg"
import Link from "next/link"
import { useAuth, sanitizeErrorMessage, ROUTES, usePublicConfigQuery } from "@/core"
import { loginSchema, apiSendVerificationEmailByEmail } from "@/domains/auth"
import { Button, Input, Field, FieldLabel, FieldError, Alert, AlertDescription } from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, Eye, EyeOff, MailCheck, CheckCircle2, Shield } from "lucide-react"
import { ZodError } from "zod"
import { useSSOProviders } from "@/features/auth/hooks/use-sso-providers"
import type { SSOProviderInfo } from "@/domains/sso"

const MAX_FAILED_ATTEMPTS = 5
const LOCKOUT_DURATION_SECONDS = 60
const SESSION_STORAGE_KEY = "vmx_login_attempts"
const SESSION_STORAGE_LOCKOUT_KEY = "vmx_login_lockout_until"

interface StoredAttemptState {
  count: number
  lockoutUntil: number | null // epoch ms
}

const getStoredAttempts = (): StoredAttemptState => {
  if (typeof window === "undefined") return { count: 0, lockoutUntil: null }
  try {
    const raw = sessionStorage.getItem(SESSION_STORAGE_KEY)
    const lockoutRaw = sessionStorage.getItem(SESSION_STORAGE_LOCKOUT_KEY)
    return {
      count: raw ? parseInt(raw, 10) : 0,
      lockoutUntil: lockoutRaw ? parseInt(lockoutRaw, 10) : null,
    }
  } catch {
    return { count: 0, lockoutUntil: null }
  }
}

const storeAttempts = (count: number, lockoutUntil: number | null) => {
  try {
    sessionStorage.setItem(SESSION_STORAGE_KEY, String(count))
    if (lockoutUntil !== null) {
      sessionStorage.setItem(SESSION_STORAGE_LOCKOUT_KEY, String(lockoutUntil))
    } else {
      sessionStorage.removeItem(SESSION_STORAGE_LOCKOUT_KEY)
    }
  } catch {
    // sessionStorage unavailable - degrade gracefully
  }
}

const clearAttempts = () => {
  try {
    sessionStorage.removeItem(SESSION_STORAGE_KEY)
    sessionStorage.removeItem(SESSION_STORAGE_LOCKOUT_KEY)
  } catch {
    // sessionStorage unavailable
  }
}

const SSO_ERROR_MESSAGES: Record<string, string> = {
  domain_not_allowed: "Your email domain is not allowed for this SSO provider.",
  account_not_found: "No account found. Contact your administrator.",
  identity_not_linked: "An account with this email exists. Link your SSO identity from account settings first.",
  account_deactivated: "Your account has been deactivated. Contact your administrator.",
  internal_error: "SSO authentication failed. Please try again.",
}

const providerIcon = (provider: string) => {
  switch (provider) {
    case "google":
      return (
        <svg className="size-4" viewBox="0 0 24 24" fill="currentColor">
          <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
          <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
          <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
          <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
        </svg>
      )
    case "github":
      return (
        <svg className="size-4" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
        </svg>
      )
    default:
      return <Shield className="size-4" />
  }
}

const SSOButtons = ({ providers, redirect }: { providers: SSOProviderInfo[]; redirect: string }) => {
  if (providers.length === 0) return null

  return (
    <div className="space-y-2">
      <div className="relative my-4">
        <div className="absolute inset-0 flex items-center">
          <span className="w-full border-t" />
        </div>
        <div className="relative flex justify-center text-xs uppercase">
          <span className="bg-card px-2 text-muted-foreground">Or continue with</span>
        </div>
      </div>
      {providers.map((provider) => (
        <Button
          key={provider.id}
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => {
            window.location.href = `/api/auth/sso/${provider.id}/login?redirect=${encodeURIComponent(redirect)}`
          }}
        >
          {providerIcon(provider.provider)}
          <span className="ml-2">{provider.displayName}</span>
        </Button>
      ))}
    </div>
  )
}

const LoginFormInner = ({ onLoginSuccess }: { onLoginSuccess?: () => void }) => {
  const { login } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()

  const { setupRequired } = usePublicConfigQuery()
  const { data: ssoProvidersRes } = useSSOProviders()

  const ssoData = ssoProvidersRes?.data
  const passwordLoginEnabled = ssoData?.passwordLoginEnabled ?? true
  const registrationEnabled = ssoData?.registrationEnabled ?? true
  const ssoProviders = ssoData?.providers ?? []

  useEffect(() => {
    if (setupRequired) {
      router.replace(ROUTES.SETUP)
    }
  }, [setupRequired, router])

  // Validate redirect is a same-origin relative path to prevent open redirect
  const rawRedirect = searchParams.get("redirect") ?? ROUTES.DASHBOARD
  const redirect = rawRedirect.startsWith("/") && !rawRedirect.startsWith("//")
    ? rawRedirect
    : ROUTES.DASHBOARD

  // Handle SSO error query params
  const ssoError = searchParams.get("error")
  const ssoMessage = searchParams.get("message")

  // Clean up SSO error params from URL after reading
  useEffect(() => {
    if (ssoError) {
      const params = new URLSearchParams(searchParams.toString())
      params.delete("error")
      params.delete("message")
      const remaining = params.toString()
      const newPath = window.location.pathname + (remaining ? `?${remaining}` : "")
      router.replace(newPath)
    }
  }, [ssoError, searchParams, router])

  // One-time banners: read from sessionStorage on mount, clear immediately
  const [showSetupBanner] = useState(() => {
    if (typeof window === "undefined") return false
    const flag = sessionStorage.getItem("vmx_just_setup") === "true"
    if (flag) sessionStorage.removeItem("vmx_just_setup")
    return flag
  })
  const [showRegisteredBanner] = useState(() => {
    if (typeof window === "undefined") return false
    const flag = sessionStorage.getItem("vmx_just_registered") === "true"
    if (flag) sessionStorage.removeItem("vmx_just_registered")
    return flag
  })

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  // Email verification state
  const [emailVerificationRequired, setEmailVerificationRequired] = useState(false)
  const [resendLoading, setResendLoading] = useState(false)
  const [resendSuccess, setResendSuccess] = useState(false)

  const [failedAttempts, setFailedAttempts] = useState(() => getStoredAttempts().count)
  const [lockoutUntil, setLockoutUntil] = useState<number | null>(
    () => getStoredAttempts().lockoutUntil
  )
  const [countdown, setCountdown] = useState(0)

  const isLockedOut = lockoutUntil !== null && countdown > 0

  // Countdown timer during lockout
  useEffect(() => {
    if (!lockoutUntil) {
      return
    }

    const tick = () => {
      const remaining = Math.max(0, Math.ceil((lockoutUntil - Date.now()) / 1000))
      setCountdown(remaining)

      if (remaining <= 0) {
        // Lockout expired - clear state
        setLockoutUntil(null)
        setFailedAttempts(0)
        storeAttempts(0, null)
      }
    }

    tick() // run immediately
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [lockoutUntil])

  const recordFailure = useCallback(() => {
    const newCount = failedAttempts + 1
    setFailedAttempts(newCount)

    if (newCount >= MAX_FAILED_ATTEMPTS) {
      const until = Date.now() + LOCKOUT_DURATION_SECONDS * 1000
      setLockoutUntil(until)
      storeAttempts(newCount, until)
    } else {
      storeAttempts(newCount, null)
    }
  }, [failedAttempts])

  const handleResendVerification = useCallback(async () => {
    if (!email || resendLoading) return
    setResendLoading(true)
    setResendSuccess(false)
    try {
      await apiSendVerificationEmailByEmail(email)
      setResendSuccess(true)
    } catch {
      // Silently handle - the server may reject for security reasons
      // but we still show success to avoid email enumeration
      setResendSuccess(true)
    } finally {
      setResendLoading(false)
    }
  }, [email, resendLoading])

  const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setErrors({})
    setServerError("")
    setEmailVerificationRequired(false)
    setResendSuccess(false)

    // Prevent submission during lockout
    if (isLockedOut) return

    try {
      const data = loginSchema.parse({ email, password })
      setLoading(true)
      const result = await login(data.email, data.password)

      // Successful login - reset throttle state
      setFailedAttempts(0)
      setLockoutUntil(null)
      clearAttempts()

      if (result?.mustChangePassword) {
        router.push(ROUTES.CHANGE_PASSWORD)
      } else if (onLoginSuccess) {
        onLoginSuccess()
      } else {
        router.push(redirect)
      }
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
      } else {
        const rawMessage = err instanceof Error ? err.message : ""
        if (rawMessage.includes("email_verification_required")) {
          setEmailVerificationRequired(true)
        } else {
          setServerError(sanitizeErrorMessage(err))
          recordFailure()
        }
      }
    } finally {
      setLoading(false)
    }
  }

  const ssoErrorMessage = ssoError
    ? ssoMessage
      ? decodeURIComponent(ssoMessage)
      : SSO_ERROR_MESSAGES[ssoError] ?? "SSO authentication failed."
    : null

  return (
    <div className="w-full max-w-sm px-4">
      <Card>
        <CardHeader className="text-center">
          <Image src={veilenceLogo} alt="Veilence-MX" width={40} height={40} className="mx-auto mb-2 size-10" />
          <CardTitle className="text-xl">Welcome back</CardTitle>
          <CardDescription>
            Sign in to your Veilence-MX account
          </CardDescription>
        </CardHeader>
        <CardContent>
          {ssoErrorMessage && (
            <Alert variant="destructive" className="mb-4 text-center bg-destructive/10 border-destructive">
              <AlertDescription>{ssoErrorMessage}</AlertDescription>
            </Alert>
          )}

          {passwordLoginEnabled && (
            <form onSubmit={handleSubmit} noValidate className="space-y-4">
              {serverError && (
                <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
                  <AlertDescription>{serverError}</AlertDescription>
                </Alert>
              )}
              {showSetupBanner && (
                <Alert variant="default" data-testid="setup-complete-banner">
                  <CheckCircle2 className="h-4 w-4" />
                  <AlertDescription>
                    Account created successfully. Please sign in.
                  </AlertDescription>
                </Alert>
              )}
              {showRegisteredBanner && !emailVerificationRequired && (
                <Alert variant="default" data-testid="registered-banner">
                  <MailCheck className="h-4 w-4" />
                  <AlertDescription>
                    Please check your email to verify your account before signing in.
                  </AlertDescription>
                </Alert>
              )}
              {isLockedOut && (
                <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive" data-testid="lockout-message">
                  <AlertDescription>
                    Too many failed login attempts. Please try again in{" "}
                    <span data-testid="lockout-countdown">{countdown}</span>{" "}
                    second{countdown !== 1 ? "s" : ""}.
                  </AlertDescription>
                </Alert>
              )}
              {emailVerificationRequired && (
                <Alert
                  variant="warning"
                  data-testid="email-verification-banner"
                >
                  <MailCheck className="h-4 w-4" />
                  <AlertDescription>
                    <div className="space-y-2">
                      <p>
                        Please verify your email address before logging in. Check your inbox for a verification link.
                      </p>
                      {resendSuccess ? (
                        <p className="text-green-700 dark:text-green-400 font-medium">
                          Verification email sent! Check your inbox.
                        </p>
                      ) : (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={handleResendVerification}
                          disabled={resendLoading}
                          className="border-amber-400 text-amber-800 hover:bg-amber-100 dark:border-amber-600 dark:text-amber-300 dark:hover:bg-amber-900"
                        >
                          {resendLoading && <Loader2 className="mr-2 h-3 w-3 animate-spin" />}
                          Resend verification email
                        </Button>
                      )}
                    </div>
                  </AlertDescription>
                </Alert>
              )}
              <Field data-invalid={!!errors.email}>
                <FieldLabel htmlFor="email">Email</FieldLabel>
                <Input
                  id="email"
                  type="email"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
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
                    placeholder="Enter your password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="current-password"
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
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={loading || isLockedOut}
              >
                {loading && <Loader2 className="mr-2 size-4 animate-spin" />}
                Sign In
              </Button>
            </form>
          )}

          <SSOButtons providers={ssoProviders} redirect={redirect} />

          {passwordLoginEnabled && registrationEnabled && (
            <div className="mt-4 text-center text-sm text-muted-foreground">
              Don&apos;t have an account?{" "}
              <Link
                href={ROUTES.REGISTER}
                className="font-medium text-primary underline-offset-4 hover:underline"
              >
                Create account
              </Link>
            </div>
          )}
          {passwordLoginEnabled && (
            <div className="mt-2 text-center">
              <Link
                href={ROUTES.FORGOT_PASSWORD}
                className="text-sm text-muted-foreground hover:text-primary underline-offset-4 hover:underline"
              >
                Forgot your password?
              </Link>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export const LoginForm = ({ onLoginSuccess }: { onLoginSuccess?: () => void }) => {
  return (
    <Suspense>
      <LoginFormInner onLoginSuccess={onLoginSuccess} />
    </Suspense>
  )
}
