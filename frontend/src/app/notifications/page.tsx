"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { NotificationsListView } from "@/features/notifications"

const NotificationsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="notifications">
      <NotificationsListView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default NotificationsPage
