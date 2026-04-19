"use client"

import { ProtectedRoute } from "@/features/auth"
import { WorkspaceDetailView } from "@/features/admin"

const WorkspaceDetailPage = () => (
  <ProtectedRoute>
    <WorkspaceDetailView />
  </ProtectedRoute>
)
export default WorkspaceDetailPage
