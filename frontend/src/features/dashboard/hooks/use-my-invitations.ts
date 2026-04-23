"use client"

import { useQuery } from "@tanstack/react-query"
import { apiGetMyInvitations } from "@/domains/admin"

export const myInvitationKeys = {
  all: ["my-invitations"] as const,
}

export const useMyInvitations = () =>
  useQuery({
    queryKey: myInvitationKeys.all,
    queryFn: () => apiGetMyInvitations(),
  })
