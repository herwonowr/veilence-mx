"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { PackagesListView } from "@/features/packages"

const PackagesPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="packages">
      <PackagesListView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default PackagesPage
