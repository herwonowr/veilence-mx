"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { StalePackagesView } from "@/features/packages"

const StalePackagesPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="packages">
      <StalePackagesView />
    </RequireOrg>
  </ProtectedRoute>
)
export default StalePackagesPage
