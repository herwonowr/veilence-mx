"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { ReleaseDetailView } from "@/features/releases"

const ReleaseDetailPage = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => (
  <ProtectedRoute>
    <RequireOrg feature="release details">
      <ReleaseDetailView params={params} />
    </RequireOrg>
  </ProtectedRoute>
)
export default ReleaseDetailPage
