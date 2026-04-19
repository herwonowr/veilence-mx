"use client"

import { ProtectedRoute } from "@/features/auth"
import { AuditLogView } from "@/features/admin"

const AuditLogPage = () => (
  <ProtectedRoute>
    <AuditLogView />
  </ProtectedRoute>
)
export default AuditLogPage
