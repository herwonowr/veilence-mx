"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { StalePackagesView } from "@/features/packages"

const StalePackagesPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="packages">
      <StalePackagesView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default StalePackagesPage
