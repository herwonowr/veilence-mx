"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { ApiKeysView } from "@/features/account"

const ApiKeysPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="API keys">
      <ApiKeysView />
    </RequireOrg>
  </ProtectedRoute>
)
export default ApiKeysPage
