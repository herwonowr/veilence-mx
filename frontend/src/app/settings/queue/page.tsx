"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { QueueView } from "@/features/settings"

const QueuePage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="queue monitor">
      <QueueView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default QueuePage
