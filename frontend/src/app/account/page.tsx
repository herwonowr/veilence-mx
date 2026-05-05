"use client"

import { Suspense, useEffect, useRef, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { toast } from "sonner"
import { ProtectedRoute } from "@/features/auth"
import { AccountView } from "@/features/account"
import { LinkedIdentities } from "@/features/sso"

const SSO_LINK_ERROR_MESSAGES: Record<string, string> = {
  sso_failed: "Failed to link SSO identity. Please try again.",
  domain_not_allowed: "Your email domain is not allowed for this SSO provider.",
  identity_already_linked: "This SSO identity is already linked to another account.",
  account_deactivated: "Your account has been deactivated. Contact your administrator.",
  internal_error: "An error occurred while linking your identity. Please try again.",
}

const AccountPageInner = () => {
  const router = useRouter()
  const searchParams = useSearchParams()

  // Capture SSO error/success on mount, then clean URL
  const [ssoResult] = useState(() => {
    const error = searchParams.get("error")
    const message = searchParams.get("message")
    const linked = searchParams.get("linked")
    if (error) return { type: "error" as const, error, message }
    if (linked) return { type: "success" as const }
    return null
  })

  const didToast = useRef(false)

  useEffect(() => {
    if (didToast.current || !ssoResult) return
    didToast.current = true

    if (ssoResult.type === "error") {
      const errorMessage = ssoResult.message
        ? decodeURIComponent(ssoResult.message)
        : SSO_LINK_ERROR_MESSAGES[ssoResult.error] ?? "Failed to link SSO identity."
      toast.error(errorMessage)
    } else if (ssoResult.type === "success") {
      toast.success("Identity linked successfully.")
    }

    // Clean URL
    const params = new URLSearchParams(searchParams.toString())
    params.delete("error")
    params.delete("message")
    params.delete("linked")
    const remaining = params.toString()
    const newPath = window.location.pathname + (remaining ? `?${remaining}` : "")
    router.replace(newPath)
  }, [ssoResult, searchParams, router])

  return (
    <ProtectedRoute>
      <div className="space-y-6">
        <AccountView />
        <LinkedIdentities />
      </div>
    </ProtectedRoute>
  )
}

const AccountPage = () => {
  return (
    <Suspense>
      <AccountPageInner />
    </Suspense>
  )
}
export default AccountPage
