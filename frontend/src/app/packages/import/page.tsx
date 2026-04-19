"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { PackageImportView } from "@/features/packages"

const PackageImportPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="package import">
      <PackageImportView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default PackageImportPage
