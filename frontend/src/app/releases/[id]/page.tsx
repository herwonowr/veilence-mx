"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { ReleaseDetailView } from "@/features/releases"

const ReleaseDetailPage = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => (
  <ProtectedRoute>
    <RequireWorkspace feature="release details">
      <ReleaseDetailView params={params} />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default ReleaseDetailPage
