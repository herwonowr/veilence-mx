import { fetchApi, config } from "@/core"
import type { ApiResponse } from "@/core"
import type {
  SSOConfig,
  SSOProvidersResponse,
  CreateSSOConfigRequest,
  UpdateSSOConfigRequest,
  TestSSOConfigResponse,
  SAMLMetadataImportResponse,
  UserIdentity,
  SPCertificateResponse,
} from "@/domains/sso/types/sso.types"

// ---------------------------------------------------------------------------
// Query keys for SSO-related React Query caching
// ---------------------------------------------------------------------------

export const ssoKeys = {
  all: ["sso"] as const,
  providers: () => [...ssoKeys.all, "providers"] as const,
  configs: () => [...ssoKeys.all, "configs"] as const,
  config: (id: string) => [...ssoKeys.all, "config", id] as const,
  identities: () => [...ssoKeys.all, "identities"] as const,
}

// ---------------------------------------------------------------------------
// Public SSO (no auth required)
// ---------------------------------------------------------------------------

/** GET /api/auth/sso/providers - get enabled SSO providers for login page */
export const apiGetSSOProviders = async (): Promise<
  ApiResponse<SSOProvidersResponse>
> => fetchApi<SSOProvidersResponse>("/api/auth/sso/providers")

// SSO login initiation is a full-page navigation, not a fetch:
// window.location.href = `/api/auth/sso/${configId}/login?redirect=/dashboard`

// ---------------------------------------------------------------------------
// Identity management (auth required)
// ---------------------------------------------------------------------------

/** GET /api/auth/identities - list linked identities for current user */
export const apiGetLinkedIdentities = async (): Promise<
  ApiResponse<UserIdentity[]>
> => fetchApi<UserIdentity[]>("/api/auth/identities")

/** Identity linking is a full-page navigation:
 *  window.location.href = buildLinkIdentityUrl(configId, callbackUrl, accessToken)
 */
export const buildLinkIdentityUrl = (configId: string, callbackUrl?: string, accessToken?: string): string => {
  const params = new URLSearchParams({ configId })
  if (callbackUrl) {
    params.set("callback_url", callbackUrl)
  }
  if (accessToken) {
    params.set("access_token", accessToken)
  }
  return `${config.apiBaseUrl}/api/auth/identities/link?${params.toString()}`
}

/** DELETE /api/auth/identities/{id} - unlink identity */
export const apiUnlinkIdentity = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/auth/identities/${id}`, { method: "DELETE" })

// ---------------------------------------------------------------------------
// Platform SSO admin (super-admin only)
// ---------------------------------------------------------------------------

/** GET /api/admin/sso - list all SSO configs */
export const apiGetPlatformSSOConfigs = async (): Promise<
  ApiResponse<SSOConfig[]>
> => fetchApi<SSOConfig[]>("/api/admin/sso")

/** GET /api/admin/sso/{id} - get SSO config detail */
export const apiGetPlatformSSOConfig = async (
  id: string
): Promise<ApiResponse<SSOConfig>> =>
  fetchApi<SSOConfig>(`/api/admin/sso/${id}`)

/** POST /api/admin/sso - create SSO config */
export const apiCreatePlatformSSOConfig = async (
  config: CreateSSOConfigRequest
): Promise<ApiResponse<SSOConfig>> =>
  fetchApi<SSOConfig>("/api/admin/sso", {
    method: "POST",
    body: JSON.stringify(config),
  })

/** PUT /api/admin/sso/{id} - update SSO config */
export const apiUpdatePlatformSSOConfig = async (
  id: string,
  config: UpdateSSOConfigRequest
): Promise<ApiResponse<SSOConfig>> =>
  fetchApi<SSOConfig>(`/api/admin/sso/${id}`, {
    method: "PUT",
    body: JSON.stringify(config),
  })

/** DELETE /api/admin/sso/{id} - delete SSO config */
export const apiDeletePlatformSSOConfig = async (
  id: string
): Promise<ApiResponse<null>> =>
  fetchApi<null>(`/api/admin/sso/${id}`, { method: "DELETE" })

/** POST /api/admin/sso/{id}/test - test SSO connectivity */
export const apiTestPlatformSSOConfig = async (
  id: string
): Promise<ApiResponse<TestSSOConfigResponse>> =>
  fetchApi<TestSSOConfigResponse>(`/api/admin/sso/${id}/test`, {
    method: "POST",
  })

/** POST /api/admin/sso/saml/import-metadata - import SAML metadata from URL */
export const apiImportSAMLMetadata = async (
  metadataUrl: string
): Promise<ApiResponse<SAMLMetadataImportResponse>> =>
  fetchApi<SAMLMetadataImportResponse>("/api/admin/sso/saml/import-metadata", {
    method: "POST",
    body: JSON.stringify({ metadataUrl }),
  })

/** GET /api/auth/saml/certificate - get SP signing certificate (public, no auth) */
export const apiGetSAMLSPCertificate = async (): Promise<
  ApiResponse<SPCertificateResponse>
> => fetchApi<SPCertificateResponse>("/api/auth/saml/certificate")
