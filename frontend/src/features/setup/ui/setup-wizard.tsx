"use client"

import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useRouter } from "next/navigation"
import Image from "next/image"
import veilenceLogo from "@/../public/veilence-mx.svg"
import { sanitizeErrorMessage, usePublicConfig, ROUTES } from "@/core"
import { useSetup } from "@/features/setup/hooks/use-setup"
import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldError,
  Alert,
  AlertDescription,
  Skeleton,
} from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, Eye, EyeOff, Rocket } from "lucide-react"
import { z, ZodError } from "zod"
import { getPasswordStrength } from "@/domains/auth"

const setupSchema = z
  .object({
    firstName: z.string().min(1, "First name is required"),
    lastName: z.string().min(1, "Last name is required"),
    email: z.email("Please enter a valid email address"),
    password: z
      .string()
      .min(8, "Password must be at least 8 characters")
      .max(72, "Password must be at most 72 characters"),
    confirmPassword: z.string().min(1, "Please confirm your password"),
    workspaceName: z
      .string()
      .min(2, "Workspace name must be at least 2 characters")
      .max(100, "Workspace name must be at most 100 characters"),
    workspaceSlug: z
      .string()
      .min(2, "Slug must be at least 2 characters")
      .max(50, "Slug must be at most 50 characters")
      .regex(
        /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
        "Slug must be lowercase letters, numbers, and hyphens"
      ),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

const toSlug = (name: string): string =>
  name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")

export const SetupWizard = () => {
  const router = useRouter()
  const { initialize, isLoading: setupLoading } = useSetup()
  const { config, isLoading: configLoading, fetchConfig } = usePublicConfig()

  const didInit = useRef(false)
  useEffect(() => {
    if (didInit.current) return
    didInit.current = true
    fetchConfig()
  }, [fetchConfig])

  // Redirect if setup is not required
  useEffect(() => {
    if (!configLoading && config && !config.setupRequired) {
      router.replace(ROUTES.LOGIN)
    }
  }, [configLoading, config, router])

  const [firstName, setFirstName] = useState("")
  const [lastName, setLastName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [workspaceName, setWorkspaceName] = useState("")
  const [workspaceSlug, setWorkspaceSlug] = useState("")
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const passwordStrength = useMemo(() => getPasswordStrength(password), [password])

  const handleWorkspaceNameChange = useCallback(
    (value: string) => {
      setWorkspaceName(value)
      if (!slugManuallyEdited) {
        setWorkspaceSlug(toSlug(value))
      }
    },
    [slugManuallyEdited]
  )

  const handleSlugChange = useCallback((value: string) => {
    setSlugManuallyEdited(true)
    setWorkspaceSlug(value)
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrors({})
    setServerError("")

    try {
      setupSchema.parse({
        firstName,
        lastName,
        email,
        password,
        confirmPassword,
        workspaceName,
        workspaceSlug,
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
      const result = await initialize({
        email,
        password,
        firstName,
        lastName,
        workspaceName,
        workspaceSlug,
      })
      if (result) {
        setPassword("")
        setConfirmPassword("")
        try { sessionStorage.setItem("vmx_just_setup", "true") } catch {}
        router.push(ROUTES.LOGIN)
      }
    } catch (err: unknown) {
      setServerError(sanitizeErrorMessage(err, "Setup failed"))
    }
  }

  if (configLoading) {
    return (
      <div className="w-full max-w-md px-4">
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <Skeleton className="h-8 w-48" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (config && !config.setupRequired) {
    return null
  }

  return (
    <div className="w-full max-w-md px-4">
      <Card>
        <CardHeader className="text-center">
          <Image
            src={veilenceLogo}
            alt="Veilence-MX"
            width={40}
            height={40}
            className="mx-auto mb-2 size-10"
          />
          <CardTitle className="text-xl">Welcome to Veilence-MX</CardTitle>
          <CardDescription>
            Set up your admin account and workspace to get started.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} noValidate className="space-y-4">
            {serverError && (
              <Alert
                variant="destructive"
                className="text-center bg-destructive/10 border-destructive"
              >
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}

            <div className="grid grid-cols-2 gap-3">
              <Field data-invalid={!!errors.firstName}>
                <FieldLabel htmlFor="firstName">First Name</FieldLabel>
                <Input
                  id="firstName"
                  placeholder="First name"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  autoComplete="given-name"
                />
                {errors.firstName && (
                  <FieldError>{errors.firstName}</FieldError>
                )}
              </Field>
              <Field data-invalid={!!errors.lastName}>
                <FieldLabel htmlFor="lastName">Last Name</FieldLabel>
                <Input
                  id="lastName"
                  placeholder="Last name"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  autoComplete="family-name"
                />
                {errors.lastName && (
                  <FieldError>{errors.lastName}</FieldError>
                )}
              </Field>
            </div>

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
                  placeholder="At least 8 characters"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                />
                {errors.password && (
                  <FieldError>{errors.password}</FieldError>
                )}
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
                  Confirm Password
                </FieldLabel>
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? "text" : "password"}
                  placeholder="Confirm password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
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

            <div className="border-t pt-4">
              <p className="text-sm font-medium text-muted-foreground mb-3">
                Workspace
              </p>
              <div className="space-y-4">
                <Field data-invalid={!!errors.workspaceName}>
                  <FieldLabel htmlFor="workspaceName">
                    Workspace Name
                  </FieldLabel>
                  <Input
                    id="workspaceName"
                    placeholder="Acme Corp"
                    value={workspaceName}
                    onChange={(e) => handleWorkspaceNameChange(e.target.value)}
                  />
                  {errors.workspaceName && (
                    <FieldError>{errors.workspaceName}</FieldError>
                  )}
                </Field>
                <Field data-invalid={!!errors.workspaceSlug}>
                  <FieldLabel htmlFor="workspaceSlug">
                    Workspace Slug
                  </FieldLabel>
                  <Input
                    id="workspaceSlug"
                    placeholder="acme-corp"
                    value={workspaceSlug}
                    onChange={(e) => handleSlugChange(e.target.value)}
                  />
                  {errors.workspaceSlug && (
                    <FieldError>{errors.workspaceSlug}</FieldError>
                  )}
                </Field>
              </div>
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={setupLoading}
            >
              {setupLoading ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <Rocket className="mr-2 size-4" />
              )}
              Complete Setup
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
