"use client"

import { ProtectedRoute } from "@/features/auth"
import { MyInvitationsView } from "@/features/admin"

const InvitationsPage = () => (
  <ProtectedRoute>
    <MyInvitationsView />
  </ProtectedRoute>
)
export default InvitationsPage
