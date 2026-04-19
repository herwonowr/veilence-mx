"use client"

import { ProtectedRoute } from "@/features/auth"
import { WorkspacesListView } from "@/features/admin"

const WorkspacesPage = () => (
  <ProtectedRoute>
    <WorkspacesListView />
  </ProtectedRoute>
)
export default WorkspacesPage
