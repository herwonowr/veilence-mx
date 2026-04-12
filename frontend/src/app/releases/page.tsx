"use client"

import { ProtectedRoute, RequireOrg } from "@/features/auth"
import { ReleasesListView } from "@/features/releases"

const ReleasesPage = () => (
  <ProtectedRoute>
    <RequireOrg feature="releases">
      <ReleasesListView />
    </RequireOrg>
  </ProtectedRoute>
)
export default ReleasesPage
