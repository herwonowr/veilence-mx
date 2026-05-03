"use client"

import { Input, Field, FieldLabel, FieldDescription, FieldError } from "@/ui"
import type { SSOProvider } from "@/domains/sso"

interface OAuthConfigFormProps {
  provider: SSOProvider
  oauthClientId: string
  oauthClientSecret: string
  googleHostedDomain: string
  githubOrgs: string
  onOauthClientIdChange: (v: string) => void
  onOauthClientSecretChange: (v: string) => void
  onGoogleHostedDomainChange: (v: string) => void
  onGithubOrgsChange: (v: string) => void
  errors?: Record<string, string>
  disabled?: boolean
  isEditing?: boolean
}

export const OAuthConfigForm = ({
  provider,
  oauthClientId,
  oauthClientSecret,
  googleHostedDomain,
  githubOrgs,
  onOauthClientIdChange,
  onOauthClientSecretChange,
  onGoogleHostedDomainChange,
  onGithubOrgsChange,
  errors = {},
  disabled = false,
  isEditing = false,
}: OAuthConfigFormProps) => (
  <div className="space-y-4">
    <h4 className="font-medium text-sm">
      {provider === "google" ? "Google OAuth" : "GitHub OAuth"} Settings
    </h4>

    <Field data-invalid={!!errors.oauthClientId}>
      <FieldLabel>Client ID</FieldLabel>
      <Input
        value={oauthClientId}
        onChange={(e) => onOauthClientIdChange(e.target.value)}
        placeholder="OAuth client ID"
        disabled={disabled}
      />
      {errors.oauthClientId && <FieldError>{errors.oauthClientId}</FieldError>}
    </Field>

    <Field data-invalid={!!errors.oauthClientSecret}>
      <FieldLabel>Client Secret</FieldLabel>
      {isEditing && (
        <FieldDescription>
          Leave blank to keep the existing secret. The secret is never displayed.
        </FieldDescription>
      )}
      <Input
        type="password"
        value={oauthClientSecret}
        onChange={(e) => onOauthClientSecretChange(e.target.value)}
        placeholder={isEditing ? "Leave blank to keep existing" : "OAuth client secret"}
        disabled={disabled}
      />
      {errors.oauthClientSecret && <FieldError>{errors.oauthClientSecret}</FieldError>}
    </Field>

    {provider === "google" && (
      <Field>
        <FieldLabel>Hosted Domain</FieldLabel>
        <FieldDescription>
          Restrict login to users from this Google Workspace domain.
        </FieldDescription>
        <Input
          value={googleHostedDomain}
          onChange={(e) => onGoogleHostedDomainChange(e.target.value)}
          placeholder="yourcompany.com"
          disabled={disabled}
        />
      </Field>
    )}

    {provider === "github" && (
      <Field>
        <FieldLabel>Required Organizations (comma-separated)</FieldLabel>
        <FieldDescription>
          Only allow users who belong to these GitHub organizations.
        </FieldDescription>
        <Input
          value={githubOrgs}
          onChange={(e) => onGithubOrgsChange(e.target.value)}
          placeholder="my-org, another-org"
          disabled={disabled}
        />
      </Field>
    )}
  </div>
)
