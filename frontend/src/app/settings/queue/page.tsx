"use client"

import { ProtectedRoute, RequireWorkspace, RequireRole } from "@/features/auth"
import { QueueView } from "@/features/settings"

const QueuePage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="queue monitor">
      <RequireRole minimumRole="admin">
        <QueueView />
      </RequireRole>
    </RequireWorkspace>
  </ProtectedRoute>
)
export default QueuePage
