"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { PackageDetailView } from "@/features/packages"

const PackageDetailPage = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => (
  <ProtectedRoute>
    <RequireWorkspace feature="package details">
      <PackageDetailView params={params} />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default PackageDetailPage
