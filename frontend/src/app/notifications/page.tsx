"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { NotificationsListView } from "@/features/notifications"

const NotificationsPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="notifications">
      <NotificationsListView />
    </RequireOrg>
  </ProtectedRoute>
)
export default NotificationsPage
