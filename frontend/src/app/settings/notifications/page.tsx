"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { ChannelsView } from "@/features/notifications"

const ChannelsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="notification channels">
      <ChannelsView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default ChannelsPage
