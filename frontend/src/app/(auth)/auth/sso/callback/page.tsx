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
      const error = searchParams.get("error")
      const message = searchParams.get("message")

      if (!error) {
        // Link flow success - no tokens needed, just redirect back with success indicator
        const safePath = redirect.startsWith("/") && !redirect.startsWith("//")
          ? redirect
          : ROUTES.DASHBOARD
        router.replace(`${safePath}?linked=true`)
        return
      }

      // Forward error params to the appropriate page
      const errorParams = new URLSearchParams({ error })
      if (message) errorParams.set("message", message)

      // If redirect points to an authenticated page (e.g. /account for link flow),
      // redirect there so the error is shown in context. Otherwise go to login.
      const isLinkFlow = redirect !== ROUTES.DASHBOARD && redirect !== ROUTES.LOGIN && redirect !== "/"
      const errorTarget = isLinkFlow ? redirect : ROUTES.LOGIN
      router.replace(`${errorTarget}?${errorParams.toString()}`)
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
