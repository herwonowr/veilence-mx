"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiGetLinkedIdentities,
  apiUnlinkIdentity,
  ssoKeys,
} from "@/domains/sso"

export const useLinkedIdentities = () =>
  useQuery({
    queryKey: ssoKeys.identities(),
    queryFn: apiGetLinkedIdentities,
  })

export const useUnlinkIdentity = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiUnlinkIdentity(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ssoKeys.identities() })
    },
  })
}
