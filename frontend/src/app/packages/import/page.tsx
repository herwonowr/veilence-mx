"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { PackageImportView } from "@/features/packages"

const PackageImportPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="package import">
      <PackageImportView />
    </RequireOrg>
  </ProtectedRoute>
)
export default PackageImportPage
