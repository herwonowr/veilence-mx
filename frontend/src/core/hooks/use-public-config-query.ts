"use client"

import { useQuery } from "@tanstack/react-query"
import { apiGetPublicConfig } from "@/domains/config"

export const publicConfigKeys = {
  all: ["public-config"] as const,
}

export const usePublicConfigQuery = () => {
  const query = useQuery({
    queryKey: publicConfigKeys.all,
    queryFn: () => apiGetPublicConfig(),
    staleTime: 5 * 60 * 1000,
  })

  return {
    config: query.data?.data ?? null,
    registrationEnabled: query.data?.data?.registrationEnabled ?? false,
    setupRequired: query.data?.data?.setupRequired ?? false,
    hasEmailDomainRestriction: query.data?.data?.hasEmailDomainRestriction ?? false,
    isLoading: query.isLoading,
    refetch: query.refetch,
  }
}
