"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { PackagesListView } from "@/features/packages"

const PackagesPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="packages">
      <PackagesListView />
    </RequireOrg>
  </ProtectedRoute>
)
export default PackagesPage
