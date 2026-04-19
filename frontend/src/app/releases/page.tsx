"use client"

import { ProtectedRoute, RequireWorkspace } from "@/features/auth"
import { ReleasesListView } from "@/features/releases"

const ReleasesPage = () => (
  <ProtectedRoute>
    <RequireWorkspace feature="releases">
      <ReleasesListView />
    </RequireWorkspace>
  </ProtectedRoute>
)
export default ReleasesPage
