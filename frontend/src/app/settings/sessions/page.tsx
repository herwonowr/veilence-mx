"use client"

import { ProtectedRoute } from "@/features/auth"
import { SessionsView } from "@/features/account"

const SessionsPage = () => (
  <ProtectedRoute>
    <SessionsView />
  </ProtectedRoute>
)
export default SessionsPage
