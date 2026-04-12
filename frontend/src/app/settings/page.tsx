"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { SettingsView } from "@/features/settings"

const SettingsPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="settings">
      <SettingsView />
    </RequireOrg>
  </ProtectedRoute>
)
export default SettingsPage
