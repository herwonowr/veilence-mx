"use client"

import { useQuery } from "@tanstack/react-query"
import { apiGetSAMLSPCertificate } from "@/domains/sso"

export const useSAMLSPCertificate = () =>
  useQuery({
    queryKey: ["sso", "saml", "sp-certificate"] as const,
    queryFn: apiGetSAMLSPCertificate,
    staleTime: 10 * 60 * 1000,
  })
