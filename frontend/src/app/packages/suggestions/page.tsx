"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { PackageSuggestionsView } from "@/features/packages"

const SuggestionsPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="packages">
      <PackageSuggestionsView />
    </RequireOrg>
  </ProtectedRoute>
)
export default SuggestionsPage
