"use client"

import { useQuery, useMutation } from "@tanstack/react-query"
import {
  apiGetInvitationByToken,
  apiAcceptInvitation,
  apiDeclineInvitationByToken,
} from "@/domains/admin"

export const invitationKeys = {
  all: ["invitation"] as const,
  byToken: (token: string) => [...invitationKeys.all, token] as const,
}

export const useInvitationByToken = (token: string) =>
  useQuery({
    queryKey: invitationKeys.byToken(token),
    queryFn: () => apiGetInvitationByToken(token),
    enabled: !!token,
    retry: false,
  })

export const useAcceptInvitation = () =>
  useMutation({
    mutationFn: ({
      workspaceId,
      token,
    }: {
      workspaceId: string
      token: string
    }) => apiAcceptInvitation(workspaceId, token),
  })

export const useDeclineInvitationByToken = () =>
  useMutation({
    mutationFn: ({
      workspaceId,
      token,
    }: {
      workspaceId: string
      token: string
    }) => apiDeclineInvitationByToken(workspaceId, token),
  })
