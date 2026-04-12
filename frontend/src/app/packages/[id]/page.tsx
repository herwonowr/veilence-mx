"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { PackageDetailView } from "@/features/packages"

const PackageDetailPage = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => (
  <ProtectedRoute>
    <RequireOrg feature="package details">
      <PackageDetailView params={params} />
    </RequireOrg>
  </ProtectedRoute>
)
export default PackageDetailPage
