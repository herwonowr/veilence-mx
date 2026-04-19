"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { PackageSuggestionsView } from "@/features/packages"

const SuggestionsPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="packages">
      <PackageSuggestionsView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default SuggestionsPage
