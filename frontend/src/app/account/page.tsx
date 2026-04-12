"use client"

import { ProtectedRoute } from "@/features/auth"
import { AccountView } from "@/features/account"

const AccountPage = () => (
  <ProtectedRoute>
    <AccountView />
  </ProtectedRoute>
)
export default AccountPage
