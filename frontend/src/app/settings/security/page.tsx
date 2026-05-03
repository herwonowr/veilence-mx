import { ProtectedRoute, RequireSuperAdmin } from "@/features/auth"
import { SSOAdminView } from "@/features/sso-admin"

const SecuritySettingsPage = () => (
  <ProtectedRoute>
    <RequireSuperAdmin>
      <SSOAdminView />
    </RequireSuperAdmin>
  </ProtectedRoute>
)

export default SecuritySettingsPage
