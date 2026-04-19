"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { SettingsView } from "@/features/settings"

const SettingsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="settings">
      <SettingsView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default SettingsPage
