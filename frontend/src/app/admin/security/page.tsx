import { ProtectedRoute, RequireSuperAdmin } from "@/features/auth"
import { SSOAdminView } from "@/features/sso-admin"

const AdminSecurityPage = () => (
  <ProtectedRoute>
    <RequireSuperAdmin>
      <SSOAdminView />
    </RequireSuperAdmin>
  </ProtectedRoute>
)

export default AdminSecurityPage
