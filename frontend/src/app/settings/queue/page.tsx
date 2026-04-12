"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { QueueView } from "@/features/settings"

const QueuePage = () => (
  <ProtectedRoute>
    <RequireOrg feature="queue monitor">
      <QueueView />
    </RequireOrg>
  </ProtectedRoute>
)
export default QueuePage
