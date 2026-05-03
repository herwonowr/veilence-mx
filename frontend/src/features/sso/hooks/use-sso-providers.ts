"use client"

import { useQuery } from "@tanstack/react-query"
import { apiGetSSOProviders, ssoKeys } from "@/domains/sso"

export const useSSOProviders = () =>
  useQuery({
    queryKey: ssoKeys.providers(),
    queryFn: apiGetSSOProviders,
    staleTime: 5 * 60 * 1000,
  })
