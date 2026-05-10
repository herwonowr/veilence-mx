"use client"

import { useQuery } from "@tanstack/react-query"
import { useAuth , usePublicConfigQuery } from "@/core"
import { apiGetMyInvitations } from "@/domains/admin"

export const onboardingKeys = {
  check: ["onboarding", "check"] as const,
}

export const useOnboardingCheck = () => {
  const { isAuthenticated, isLoading, workspaces, workspacesLoading, user } = useAuth()
  const { registrationEnabled } = usePublicConfigQuery()

  const {
    data: invitationsResponse,
    isLoading: invitationsLoading,
  } = useQuery({
    queryKey: onboardingKeys.check,
    queryFn: () => apiGetMyInvitations(),
    enabled: registrationEnabled && isAuthenticated && !isLoading && !workspacesLoading && workspaces.length === 0,
    staleTime: 30 * 1000,
  })

  const invitations = invitationsResponse?.data ?? []

  const isChecking =
    isLoading ||
    workspacesLoading ||
    (isAuthenticated && workspaces.length === 0 && invitationsLoading)

  const shouldShowOnboarding =
    isAuthenticated &&
    !isLoading &&
    !workspacesLoading &&
    !invitationsLoading &&
    !user?.mustChangePassword &&
    workspaces.length === 0 &&
    invitations.length === 0

  return { shouldShowOnboarding, isChecking }
}
