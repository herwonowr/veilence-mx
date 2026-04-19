"use client"

import { ProtectedRoute, RequireRole } from "@/features/auth"
import { AuditLogView } from "@/features/admin"

const AuditLogPage = () => (
  <ProtectedRoute>
    <RequireRole minimumRole="admin">
      <AuditLogView />
    </RequireRole>
  </ProtectedRoute>
)
export default AuditLogPage
