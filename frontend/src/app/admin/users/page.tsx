"use client"

import { ProtectedRoute, RequireSuperAdmin } from "@/features/auth"
import { PlatformUsersList } from "@/features/platform-users"

const AdminUsersContent = () => (
  <div className="space-y-6">
    <div>
      <h1 className="text-3xl font-bold">Users</h1>
      <p className="text-muted-foreground">
        Manage platform users, super admin privileges, and account status.
      </p>
    </div>
    <PlatformUsersList />
  </div>
)

const AdminUsersPage = () => (
  <ProtectedRoute>
    <RequireSuperAdmin>
      <AdminUsersContent />
    </RequireSuperAdmin>
  </ProtectedRoute>
)

export default AdminUsersPage
