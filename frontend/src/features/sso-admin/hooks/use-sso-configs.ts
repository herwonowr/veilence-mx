"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  apiGetPlatformSSOConfigs,
  apiGetPlatformSSOConfig,
  apiCreatePlatformSSOConfig,
  apiUpdatePlatformSSOConfig,
  apiDeletePlatformSSOConfig,
  apiTestPlatformSSOConfig,
  apiImportSAMLMetadata,
  ssoKeys,
} from "@/domains/sso"
import type { CreateSSOConfigRequest, UpdateSSOConfigRequest } from "@/domains/sso"
import { toast } from "sonner"

export const usePlatformSSOConfigs = () =>
  useQuery({
    queryKey: ssoKeys.configs(),
    queryFn: () => apiGetPlatformSSOConfigs(),
  })

export const usePlatformSSOConfig = (id: string) =>
  useQuery({
    queryKey: ssoKeys.config(id),
    queryFn: () => apiGetPlatformSSOConfig(id),
    enabled: !!id,
  })

export const useCreatePlatformSSOConfig = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (ssoConfig: CreateSSOConfigRequest) =>
      apiCreatePlatformSSOConfig(ssoConfig),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ssoKeys.configs() })
      queryClient.invalidateQueries({ queryKey: ssoKeys.providers() })
      toast.success("SSO configuration created")
    },
    onError: () => {
      toast.error("Failed to create SSO configuration")
    },
  })
}

export const useUpdatePlatformSSOConfig = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ssoConfig,
    }: {
      id: string
      ssoConfig: UpdateSSOConfigRequest
    }) => apiUpdatePlatformSSOConfig(id, ssoConfig),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ssoKeys.configs() })
      queryClient.invalidateQueries({ queryKey: ssoKeys.providers() })
      queryClient.invalidateQueries({
        queryKey: ssoKeys.config(variables.id),
      })
      toast.success("SSO configuration updated")
    },
    onError: () => {
      toast.error("Failed to update SSO configuration")
    },
  })
}

export const useDeletePlatformSSOConfig = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiDeletePlatformSSOConfig(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ssoKeys.configs() })
      queryClient.invalidateQueries({ queryKey: ssoKeys.providers() })
      queryClient.invalidateQueries({ queryKey: ssoKeys.identities() })
      toast.success("SSO configuration deleted")
    },
    onError: () => {
      toast.error("Failed to delete SSO configuration")
    },
  })
}

export const useTestPlatformSSOConfig = () =>
  useMutation({
    mutationFn: (id: string) => apiTestPlatformSSOConfig(id),
    onSuccess: (response) => {
      if (response.data?.success) {
        toast.success(response.data.message || "SSO connectivity test passed")
      } else {
        toast.error(response.data?.message || "SSO connectivity test failed")
      }
    },
    onError: () => {
      toast.error("Failed to test SSO configuration")
    },
  })

export const useImportSAMLMetadata = () =>
  useMutation({
    mutationFn: (metadataUrl: string) => apiImportSAMLMetadata(metadataUrl),
    onSuccess: () => {
      toast.success("SAML metadata imported successfully")
    },
    onError: () => {
      toast.error("Failed to import SAML metadata from URL")
    },
  })
