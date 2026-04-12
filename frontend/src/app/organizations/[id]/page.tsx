"use client"

import { ProtectedRoute } from "@/features/auth"
import { OrganizationDetailView } from "@/features/admin"

const OrgDetailPage = () => (
  <ProtectedRoute>
    <OrganizationDetailView />
  </ProtectedRoute>
)
export default OrgDetailPage
