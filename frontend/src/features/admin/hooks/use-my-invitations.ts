"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiGetMyInvitations,
  apiAcceptInvitationById,
  apiDeclineInvitationById,
} from "@/domains/admin"
import { toast } from "sonner"
import { sanitizeErrorMessage } from "@/core"

export const myInvitationKeys = {
  all: ["my-invitations"] as const,
}

export const useMyInvitations = () =>
  useQuery({
    queryKey: myInvitationKeys.all,
    queryFn: () => apiGetMyInvitations(),
  })

export const useAcceptInvitationById = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiAcceptInvitationById(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: myInvitationKeys.all })
      toast.success("Invitation accepted")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to accept invitation"))
    },
  })
}

export const useDeclineInvitationById = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiDeclineInvitationById(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: myInvitationKeys.all })
      toast.success("Invitation declined")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to decline invitation"))
    },
  })
}
