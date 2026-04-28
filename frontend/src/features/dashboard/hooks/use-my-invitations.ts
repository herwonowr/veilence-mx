"use client"

import { useQuery } from "@tanstack/react-query"
import { apiGetMyInvitations } from "@/domains/admin"
import { usePublicConfigQuery } from "@/core"

export const myInvitationKeys = {
  all: ["my-invitations"] as const,
}

export const useMyInvitations = () => {
  const { registrationEnabled } = usePublicConfigQuery()

  return useQuery({
    queryKey: myInvitationKeys.all,
    queryFn: () => apiGetMyInvitations(),
    enabled: registrationEnabled,
  })
}
