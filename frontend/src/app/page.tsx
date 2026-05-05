"use client"

import { Suspense, useEffect } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { ProtectedRoute } from "@/features/auth"
import { DashboardView } from "@/features/dashboard"
import { useAuth, ROUTES } from "@/core"

const SSOErrorForwarder = () => {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { isAuthenticated, isLoading } = useAuth()

  useEffect(() => {
    const error = searchParams.get("error")
    if (error && !isLoading && !isAuthenticated) {
      const message = searchParams.get("message")
      const params = new URLSearchParams({ error })
      if (message) params.set("message", message)
      router.replace(`${ROUTES.LOGIN}?${params.toString()}`)
    }
  }, [searchParams, isLoading, isAuthenticated, router])

  return null
}

const DashboardPage = () => (
  <>
    <Suspense>
      <SSOErrorForwarder />
    </Suspense>
    <ProtectedRoute>
      <DashboardView />
    </ProtectedRoute>
  </>
)
export default DashboardPage
