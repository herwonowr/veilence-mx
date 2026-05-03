// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type SSOProvider = "saml" | "google" | "github"
export type AuthProvider = "local" | "saml" | "google" | "github"

// ---------------------------------------------------------------------------
// Read models
// ---------------------------------------------------------------------------

export interface SSOConfig {
  id: string
  provider: SSOProvider
  displayName: string
  isEnabled: boolean
  autoCreateUser: boolean
  allowedDomains: string[]

  // SAML
  samlEntityId?: string
  samlSsoUrl?: string
  samlCertificate?: string
  samlAttrEmail?: string
  samlAttrFirstName?: string
  samlAttrLastName?: string

  // OAuth
  oauthClientId?: string
  googleHostedDomain?: string
  githubOrgs?: string[]

  createdAt: string
  updatedAt: string
}

export interface UserIdentity {
  id: string
  provider: SSOProvider
  providerUserId: string
  providerEmail: string
  linkedAt: string
}

// ---------------------------------------------------------------------------
// Public SSO providers (login page)
// ---------------------------------------------------------------------------

export interface SSOProvidersResponse {
  passwordLoginEnabled: boolean
  registrationEnabled: boolean
  providers: SSOProviderInfo[]
}

export interface SSOProviderInfo {
  id: string
  provider: SSOProvider
  displayName: string
}

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

/** Body for POST /api/admin/sso */
export interface CreateSSOConfigRequest {
  provider: SSOProvider
  displayName: string
  isEnabled: boolean
  autoCreateUser: boolean
  allowedDomains: string[]

  // SAML
  samlEntityId?: string
  samlSsoUrl?: string
  samlCertificate?: string
  samlAttrEmail?: string
  samlAttrFirstName?: string
  samlAttrLastName?: string

  // OAuth
  oauthClientId?: string
  oauthClientSecret?: string
  googleHostedDomain?: string
  githubOrgs?: string[]
}

/** Body for PUT /api/admin/sso/{id} */
export interface UpdateSSOConfigRequest {
  displayName?: string
  isEnabled?: boolean
  autoCreateUser?: boolean
  allowedDomains?: string[]

  // SAML
  samlEntityId?: string
  samlSsoUrl?: string
  samlCertificate?: string
  samlAttrEmail?: string
  samlAttrFirstName?: string
  samlAttrLastName?: string

  // OAuth
  oauthClientId?: string
  oauthClientSecret?: string
  googleHostedDomain?: string
  githubOrgs?: string[]
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

/** Response from POST /api/admin/sso/{id}/test */
export interface TestSSOConfigResponse {
  success: boolean
  message: string
  idpEntityId?: string
  idpSsoUrl?: string
}

/** Response from POST /api/admin/sso/saml/import-metadata */
export interface SAMLMetadataImportResponse {
  entityId: string
  ssoUrl: string
  certificate: string
  sloUrl?: string
}
