"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { ApiKeysView } from "@/features/account"

const ApiKeysPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="API keys">
      <ApiKeysView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default ApiKeysPage
