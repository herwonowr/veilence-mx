"use client"

import { useParams } from "next/navigation"
import { ProtectedRoute, RequireRole } from "@/features/auth"
import { AuditLogView } from "@/features/admin"

const AuditLogPage = () => {
  const params = useParams<{ id: string }>()
  return (
    <ProtectedRoute>
      <RequireRole minimumRole="admin" workspaceId={params.id}>
        <AuditLogView />
      </RequireRole>
    </ProtectedRoute>
  )
}
export default AuditLogPage
