"use client"

import { ProtectedRoute } from "@/features/auth"
import { AccountView } from "@/features/account"
import { LinkedIdentities } from "@/features/sso"

const AccountPage = () => {
  return (
    <ProtectedRoute>
      <AccountView />
      <LinkedIdentities />
    </ProtectedRoute>
  )
}
export default AccountPage
