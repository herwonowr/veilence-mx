"use client"

import { ProtectedRoute, RequireWorkspace, RequireRole } from "@/features/auth"
import { SettingsView } from "@/features/settings"

const SettingsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="settings">
      <RequireRole minimumRole="admin">
        <SettingsView />
      </RequireRole>
    </RequireWorkspace>
  </ProtectedRoute>
)
export default SettingsPage
