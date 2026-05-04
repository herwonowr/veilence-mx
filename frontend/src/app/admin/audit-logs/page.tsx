"use client"

import { ProtectedRoute, RequireSuperAdmin } from "@/features/auth"
import { PlatformAuditLogsList } from "@/features/platform-audit-logs"

const AdminAuditLogsContent = () => (
  <div className="space-y-6">
    <div>
      <h1 className="text-3xl font-bold">Audit Logs</h1>
      <p className="text-muted-foreground">
        View activity history across all workspaces on the platform.
      </p>
    </div>
    <PlatformAuditLogsList />
  </div>
)

const AdminAuditLogsPage = () => (
  <ProtectedRoute>
    <RequireSuperAdmin>
      <AdminAuditLogsContent />
    </RequireSuperAdmin>
  </ProtectedRoute>
)

export default AdminAuditLogsPage
