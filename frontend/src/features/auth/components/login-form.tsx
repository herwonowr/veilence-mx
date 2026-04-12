"use client"

import { Suspense, useCallback, useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Link from "next/link"
import { useAuth } from "@/lib/auth-context"
import { loginSchema } from "@/lib/validations"
import { sanitizeErrorMessage } from "@/lib/error-sanitizer"
import { Button } from "@/components/ui/button"
import { FormField } from "@/components/form-field"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Shield, Loader2, Eye, EyeOff } from "lucide-react"
import { ZodError } from "zod"

// SEC-S4-10: Login throttling constants
const MAX_FAILED_ATTEMPTS = 5
const LOCKOUT_DURATION_SECONDS = 60
const SESSION_STORAGE_KEY = "vmx_login_attempts"
const SESSION_STORAGE_LOCKOUT_KEY = "vmx_login_lockout_until"

interface StoredAttemptState {
  count: number
  lockoutUntil: number | null // epoch ms
}

function getStoredAttempts(): StoredAttemptState {
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

function storeAttempts(count: number, lockoutUntil: number | null) {
  try {
    sessionStorage.setItem(SESSION_STORAGE_KEY, String(count))
    if (lockoutUntil !== null) {
      sessionStorage.setItem(SESSION_STORAGE_LOCKOUT_KEY, String(lockoutUntil))
    } else {
      sessionStorage.removeItem(SESSION_STORAGE_LOCKOUT_KEY)
    }
  } catch {
    // sessionStorage unavailable — degrade gracefully
  }
}

function clearAttempts() {
  try {
    sessionStorage.removeItem(SESSION_STORAGE_KEY)
    sessionStorage.removeItem(SESSION_STORAGE_LOCKOUT_KEY)
  } catch {
    // sessionStorage unavailable
  }
}

function LoginFormInner() {
  const { login } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()

  // SEC-S3-004: Validate redirect is a same-origin relative path to prevent open redirect
  const rawRedirect = searchParams.get("redirect") ?? "/"
  const redirect = rawRedirect.startsWith("/") && !rawRedirect.startsWith("//")
    ? rawRedirect
    : "/"

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  // SEC-S4-10: Login attempt throttling
  const [failedAttempts, setFailedAttempts] = useState(() => getStoredAttempts().count)
  const [lockoutUntil, setLockoutUntil] = useState<number | null>(
    () => getStoredAttempts().lockoutUntil
  )
  const [countdown, setCountdown] = useState(0)

  const isLockedOut = lockoutUntil !== null && Date.now() < lockoutUntil

  // Countdown timer during lockout
  useEffect(() => {
    if (!lockoutUntil) {
      setCountdown(0)
      return
    }

    const tick = () => {
      const remaining = Math.max(0, Math.ceil((lockoutUntil - Date.now()) / 1000))
      setCountdown(remaining)

      if (remaining <= 0) {
        // Lockout expired — clear state
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrors({})
    setServerError("")

    // Prevent submission during lockout
    if (isLockedOut) return

    try {
      const data = loginSchema.parse({ email, password })
      setLoading(true)
      await login(data.email, data.password)

      // Successful login — reset throttle state
      setFailedAttempts(0)
      setLockoutUntil(null)
      clearAttempts()

      router.push(redirect)
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
        setServerError(sanitizeErrorMessage(err))
        recordFailure()
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full max-w-sm px-4">
      <Card>
        <CardHeader className="text-center">
          <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Shield className="size-5" />
          </div>
          <CardTitle className="text-xl">Welcome back</CardTitle>
          <CardDescription>
            Sign in to your Veilence-MX account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {serverError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive" role="alert">
                {serverError}
              </div>
            )}
            {isLockedOut && (
              <div
                className="rounded-md bg-destructive/10 p-3 text-sm text-destructive"
                data-testid="lockout-message"
                role="alert"
              >
                Too many failed login attempts. Please try again in{" "}
                <span data-testid="lockout-countdown">{countdown}</span>{" "}
                second{countdown !== 1 ? "s" : ""}.
              </div>
            )}
            <FormField
              id="email"
              label="Email"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              error={errors.email}
              required
              autoComplete="email"
            />
            <div className="relative">
              <FormField
                id="password"
                label="Password"
                type={showPassword ? "text" : "password"}
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                error={errors.password}
                required
                autoComplete="current-password"
              />
              <button
                type="button"
                className="absolute right-2 top-[30px] text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setShowPassword((prev) => !prev)}
                aria-label={showPassword ? "Hide password" : "Show password"}
                tabIndex={-1}
              >
                {showPassword ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </button>
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
          <div className="mt-4 text-center text-sm text-muted-foreground">
            Don&apos;t have an account?{" "}
            <Link
              href="/register"
              className="font-medium text-primary underline-offset-4 hover:underline"
            >
              Create account
            </Link>
          </div>
          <div className="mt-2 text-center">
            <Link
              href="/forgot-password"
              className="text-sm text-muted-foreground hover:text-primary underline-offset-4 hover:underline"
            >
              Forgot your password?
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export function LoginForm() {
  return (
    <Suspense>
      <LoginFormInner />
    </Suspense>
  )
}
