"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { AlertsListView } from "@/features/alerts"

const AlertsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="alerts">
      <AlertsListView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default AlertsPage
