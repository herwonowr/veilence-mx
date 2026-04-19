"use client"

import { ProtectedRoute, RequireWorkspace, RequireRole } from "@/features/auth"
import { ChannelsView } from "@/features/notifications"

const ChannelsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="notification channels">
      <RequireRole minimumRole="admin">
        <ChannelsView />
      </RequireRole>
    </RequireWorkspace>
  </ProtectedRoute>
)
export default ChannelsPage
