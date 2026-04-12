"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { AlertsListView } from "@/features/alerts"

const AlertsPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="alerts">
      <AlertsListView />
    </RequireOrg>
  </ProtectedRoute>
)
export default AlertsPage
