"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { AlertDetailView } from "@/features/alerts"

const AlertDetailPage = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => (
  <ProtectedRoute>
    <RequireWorkspace feature="alert details">
      <AlertDetailView params={params} />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default AlertDetailPage
