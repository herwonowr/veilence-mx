"use client"

import { useCallback } from "react"
import { useRouter } from "next/navigation"
import { LoginForm } from "@/features/auth"
import { ROUTES } from "@/core"

const LoginPage = () => {
  const router = useRouter()

  const handleLoginSuccess = useCallback(() => {
    router.push(ROUTES.DASHBOARD)
  }, [router])

  return <LoginForm onLoginSuccess={handleLoginSuccess} />
}

export default LoginPage
