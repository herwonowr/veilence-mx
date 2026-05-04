"use client"

import { useEffect, useRef } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { storeTokens, ROUTES } from "@/core"
import { Loader2 } from "lucide-react"

const SSOCallbackPage = () => {
  const router = useRouter()
  const searchParams = useSearchParams()
  const didProcess = useRef(false)

  useEffect(() => {
    if (didProcess.current) return
    didProcess.current = true

    const accessToken = searchParams.get("access_token")
    const refreshToken = searchParams.get("refresh_token")
    const redirect = searchParams.get("redirect") || ROUTES.DASHBOARD

    if (!accessToken || !refreshToken) {
      // Forward error params from backend to login page
      const error = searchParams.get("error") || "sso_failed"
      const message = searchParams.get("message")
      const loginParams = new URLSearchParams({ error })
      if (message) loginParams.set("message", message)
      router.replace(`${ROUTES.LOGIN}?${loginParams.toString()}`)
      return
    }

    storeTokens(accessToken, refreshToken)

    // Full page navigation to force app re-initialization with fresh auth state.
    const safePath = redirect.startsWith("/") && !redirect.startsWith("//")
      ? redirect
      : ROUTES.DASHBOARD
    window.location.href = safePath
  }, [searchParams, router])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-3">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Completing sign in...</p>
      </div>
    </div>
  )
}

export default SSOCallbackPage
