"use client"

import { ProtectedRoute } from "@/features/auth"
import { OrganizationsListView } from "@/features/admin"

const OrganizationsPage = () => (
  <ProtectedRoute>
    <OrganizationsListView />
  </ProtectedRoute>
)
export default OrganizationsPage
