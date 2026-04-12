"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { AlertDetailView } from "@/features/alerts"

const AlertDetailPage = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => (
  <ProtectedRoute>
    <RequireOrg feature="alert details">
      <AlertDetailView params={params} />
    </RequireOrg>
  </ProtectedRoute>
)
export default AlertDetailPage
