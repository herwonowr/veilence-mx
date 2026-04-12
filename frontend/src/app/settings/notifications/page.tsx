"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { ChannelsView } from "@/features/notifications"

const ChannelsPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="notification channels">
      <ChannelsView />
    </RequireOrg>
  </ProtectedRoute>
)
export default ChannelsPage
