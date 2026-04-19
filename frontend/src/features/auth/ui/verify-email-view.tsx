"use client"

import { Suspense, useEffect, useState } from "react"
import { useSearchParams } from "next/navigation"
import Link from "next/link"
import { apiVerifyEmail } from "@/domains/auth"
import { Button } from "@/ui"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui"
import { Loader2, CheckCircle2, XCircle, ArrowLeft } from "lucide-react"

type VerifyState = "loading" | "success" | "error" | "no-token"

const VerifyEmailViewInner = () => {
  const searchParams = useSearchParams()
  const token = searchParams.get("token")

  const [state, setState] = useState<VerifyState>(token ? "loading" : "no-token")
  const [errorMessage, setErrorMessage] = useState("")

  useEffect(() => {
    if (!token) return

    let cancelled = false

    const verify = async () => {
      try {
        await apiVerifyEmail(token)
        if (!cancelled) setState("success")
      } catch (err) {
        if (!cancelled) {
          setState("error")
          setErrorMessage(
            err instanceof Error
              ? err.message
              : "Failed to verify email. The link may be invalid or expired."
          )
        }
      }
    }

    verify()

    return () => {
      cancelled = true
    }
  }, [token])

  if (state === "loading") {
    return (
      <div className="w-full max-w-sm px-4">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Loader2 className="size-5 animate-spin" />
            </div>
            <CardTitle className="text-xl">Verifying your email</CardTitle>
            <CardDescription>
              Please wait while we verify your email address...
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  if (state === "success") {
    return (
      <div className="w-full max-w-sm px-4">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-green-500/10 text-green-600">
              <CheckCircle2 className="size-5" />
            </div>
            <CardTitle className="text-xl">Email verified</CardTitle>
            <CardDescription>
              Your email has been verified successfully. You can now sign in.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Link href="/login" className="block">
              <Button className="w-full">Go to sign in</Button>
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
          <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-destructive/10 text-destructive">
            <XCircle className="size-5" />
          </div>
          <CardTitle className="text-xl">
            {state === "no-token" ? "Invalid verification link" : "Verification failed"}
          </CardTitle>
          <CardDescription>
            {state === "no-token"
              ? "This verification link is missing a token. Please check the link from your email."
              : errorMessage}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <Link href="/login" className="block">
              <Button className="w-full" variant="outline">
                <ArrowLeft className="mr-2 size-3" />
                Back to sign in
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export const VerifyEmailView = () => (
  <Suspense>
    <VerifyEmailViewInner />
  </Suspense>
)
