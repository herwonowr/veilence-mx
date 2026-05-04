"use client"

import { ProtectedRoute } from "@/features/auth"
import { AccountView } from "@/features/account"
import { LinkedIdentities } from "@/features/sso"

const AccountPage = () => {
  return (
    <ProtectedRoute>
      <div className="space-y-6">
        <AccountView />
        <LinkedIdentities />
      </div>
    </ProtectedRoute>
  )
}
export default AccountPage
