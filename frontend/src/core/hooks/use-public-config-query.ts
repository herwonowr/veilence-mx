"use client"

import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { fetchApi, type ApiResponse } from "@/core/http"

export interface PublicConfig {
  registrationEnabled: boolean
  setupRequired: boolean
  hasEmailDomainRestriction: boolean
  ssoEnabled: boolean
  enabledEcosystems: string[]
}

export const publicConfigKeys = {
  all: ["public-config"] as const,
}

/** Map backend config ecosystem identifiers to frontend Ecosystem values.
 *  The config endpoint returns "pypi" but the entity/package API uses "python". */
const normalizeEcosystem = (raw: string): string => {
  if (raw === "pypi") return "python"
  return raw
}

export const usePublicConfigQuery = () => {
  const query = useQuery({
    queryKey: publicConfigKeys.all,
    queryFn: (): Promise<ApiResponse<PublicConfig>> =>
      fetchApi<PublicConfig>("/api/config/public", { skipAuth: true }),
    staleTime: 5 * 60 * 1000,
  })

  const enabledEcosystems = useMemo(
    () => (query.data?.data?.enabledEcosystems ?? []).map(normalizeEcosystem),
    [query.data?.data?.enabledEcosystems],
  )

  return {
    config: query.data?.data ?? null,
    registrationEnabled: query.data?.data?.registrationEnabled ?? false,
    setupRequired: query.data?.data?.setupRequired ?? false,
    hasEmailDomainRestriction: query.data?.data?.hasEmailDomainRestriction ?? false,
    ssoEnabled: query.data?.data?.ssoEnabled ?? false,
    enabledEcosystems,
    isLoading: query.isLoading,
    refetch: query.refetch,
  }
}
