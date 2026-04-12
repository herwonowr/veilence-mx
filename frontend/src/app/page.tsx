"use client"

import { ProtectedRoute } from "@/features/auth"
import { DashboardView } from "@/features/dashboard"

const DashboardPage = () => (
  <ProtectedRoute>
    <DashboardView />
  </ProtectedRoute>
)
export default DashboardPage
