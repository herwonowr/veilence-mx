"use client"

import { useState } from "react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Button,
  Input,
  Field,
  FieldLabel,
  FieldContent,
  FieldTitle,
  FieldDescription,
  FieldError,
  Switch,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Alert,
  AlertDescription,
} from "@/ui"
import { Save, Loader2 } from "lucide-react"
import { sanitizeErrorMessage, parseFieldErrors } from "@/core"
import type { SSOConfig, SSOProvider, CreateSSOConfigRequest, UpdateSSOConfigRequest } from "@/domains/sso"
import {
  samlConfigSchema,
  oauthConfigSchema,
  samlConfigUpdateSchema,
  oauthConfigUpdateSchema,
} from "@/domains/sso"
import { SAMLConfigForm } from "@/features/sso-admin/ui/saml-config-form"
import { OAuthConfigForm } from "@/features/sso-admin/ui/oauth-config-form"
import { SAMLSPInfo } from "@/features/sso-admin/ui/saml-sp-info"

interface SSOConfigFormProps {
  existingConfig?: SSOConfig | null
  configuredProviders?: SSOProvider[]
  isPending: boolean
  onSubmit: (data: CreateSSOConfigRequest | UpdateSSOConfigRequest) => void
  onCancel: () => void
}

const PROVIDER_LABELS: Record<SSOProvider, string> = {
  saml: "SAML 2.0",
  google: "Google OAuth",
  github: "GitHub OAuth",
}

export const SSOConfigForm = ({
  existingConfig,
  configuredProviders = [],
  isPending,
  onSubmit,
  onCancel,
}: SSOConfigFormProps) => {
  const isEditing = !!existingConfig

  const allProviders: SSOProvider[] = ["saml", "google", "github"]
  const availableProviders = allProviders.filter((p) => !configuredProviders.includes(p))

  const [provider, setProvider] = useState<SSOProvider>(
    existingConfig?.provider ?? availableProviders[0] ?? "saml"
  )
  const [displayName, setDisplayName] = useState(
    existingConfig?.displayName ?? ""
  )
  const [isEnabled, setIsEnabled] = useState(existingConfig?.isEnabled ?? true)
  const [autoCreateUser, setAutoCreateUser] = useState(
    existingConfig?.autoCreateUser ?? false
  )
  const [allowedDomains, setAllowedDomains] = useState(
    existingConfig?.allowedDomains.join(", ") ?? ""
  )

  // SAML fields
  const [samlEntityId, setSamlEntityId] = useState(existingConfig?.samlEntityId ?? "")
  const [samlSsoUrl, setSamlSsoUrl] = useState(existingConfig?.samlSsoUrl ?? "")
  const [samlCertificate, setSamlCertificate] = useState(
    existingConfig?.samlCertificate ?? ""
  )
  const [samlAttrEmail, setSamlAttrEmail] = useState(
    existingConfig?.samlAttrEmail ?? "email"
  )
  const [samlAttrFirstName, setSamlAttrFirstName] = useState(
    existingConfig?.samlAttrFirstName ?? "firstName"
  )
  const [samlAttrLastName, setSamlAttrLastName] = useState(
    existingConfig?.samlAttrLastName ?? "lastName"
  )

  // OAuth fields
  const [oauthClientId, setOauthClientId] = useState(
    existingConfig?.oauthClientId ?? ""
  )
  const [oauthClientSecret, setOauthClientSecret] = useState("")
  const [googleHostedDomain, setGoogleHostedDomain] = useState(
    existingConfig?.googleHostedDomain ?? ""
  )
  const [githubOrgs, setGithubOrgs] = useState(
    (existingConfig?.githubOrgs ?? []).join(", ")
  )

  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formSubmitted, setFormSubmitted] = useState(false)
  const [serverError, setServerError] = useState("")

  const validate = () => {
    if (!formSubmitted) return
    const formData = {
      displayName,
      isEnabled,
      autoCreateUser,
      allowedDomains,
      ...(provider === "saml"
        ? { provider: "saml" as const, samlEntityId, samlSsoUrl, samlCertificate, samlAttrEmail, samlAttrFirstName, samlAttrLastName }
        : { provider, oauthClientId, oauthClientSecret, googleHostedDomain, githubOrgs }),
    }
    const schema = isEditing
      ? (provider === "saml" ? samlConfigUpdateSchema : oauthConfigUpdateSchema)
      : (provider === "saml" ? samlConfigSchema : oauthConfigSchema)
    const result = schema.safeParse(formData)
    setErrors(result.success ? {} : parseFieldErrors(result.error))
  }

  const handleSubmit = (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault()
    setFormSubmitted(true)
    setErrors({})
    setServerError("")

    const formData = {
      displayName,
      isEnabled,
      autoCreateUser,
      allowedDomains,
      ...(provider === "saml"
        ? {
            provider: "saml" as const,
            samlEntityId,
            samlSsoUrl,
            samlCertificate,
            samlAttrEmail,
            samlAttrFirstName,
            samlAttrLastName,
          }
        : {
            provider,
            oauthClientId,
            oauthClientSecret,
            googleHostedDomain,
            githubOrgs,
          }),
    }

    try {
      if (isEditing) {
        if (provider === "saml") {
          samlConfigUpdateSchema.parse(formData)
        } else {
          oauthConfigUpdateSchema.parse(formData)
        }
      } else {
        if (provider === "saml") {
          samlConfigSchema.parse(formData)
        } else {
          oauthConfigSchema.parse(formData)
        }
      }
    } catch (err) {
      setErrors(parseFieldErrors(err))
      return
    }

    // Build the request payload
    const domains = allowedDomains
      .split(",")
      .map((d) => d.trim())
      .filter(Boolean)

    const base = {
      displayName,
      isEnabled,
      autoCreateUser,
      allowedDomains: domains,
    }

    try {
      if (provider === "saml") {
        onSubmit({
          ...base,
          ...(isEditing ? {} : { provider }),
          samlEntityId: samlEntityId || undefined,
          samlSsoUrl: samlSsoUrl || undefined,
          samlCertificate: samlCertificate || undefined,
          samlAttrEmail,
          samlAttrFirstName,
          samlAttrLastName,
        })
      } else {
        const orgs = githubOrgs
          .split(",")
          .map((o) => o.trim())
          .filter(Boolean)

        onSubmit({
          ...base,
          ...(isEditing ? {} : { provider }),
          oauthClientId: oauthClientId || undefined,
          oauthClientSecret: oauthClientSecret || undefined,
          googleHostedDomain: googleHostedDomain || undefined,
          githubOrgs: orgs,
        })
      }
    } catch (err: unknown) {
      setServerError(sanitizeErrorMessage(err, "Failed to save SSO configuration"))
    }
  }

  return (
    <>
    {provider === "saml" && (
      <SAMLSPInfo configId={existingConfig?.id} />
    )}
    <Card>
      <CardHeader>
        <CardTitle>{isEditing ? "Edit" : "Create"} SSO Configuration</CardTitle>
        <CardDescription>
          {isEditing
            ? "Update the SSO provider settings."
            : "Configure a new SSO provider for the platform."}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} noValidate className="space-y-6">
          {serverError && (
            <Alert variant="destructive" className="text-center bg-destructive/10 border-destructive">
              <AlertDescription>{serverError}</AlertDescription>
            </Alert>
          )}

          {!isEditing && (
            <Field>
              <FieldLabel>Provider</FieldLabel>
              <Select value={provider} onValueChange={(v) => setProvider(v as SSOProvider)} disabled={!isEnabled}>
                <SelectTrigger disabled={!isEnabled}>
                  <SelectValue>{PROVIDER_LABELS[provider]}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="saml" disabled={configuredProviders.includes("saml")}>SAML 2.0</SelectItem>
                  <SelectItem value="google" disabled={configuredProviders.includes("google")}>Google OAuth</SelectItem>
                  <SelectItem value="github" disabled={configuredProviders.includes("github")}>GitHub OAuth</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          )}

          <Field data-invalid={!!errors.displayName}>
            <FieldLabel>Display Name</FieldLabel>
            <Input
              value={displayName}
              onChange={(e) => { setDisplayName(e.target.value); validate() }}
              placeholder="e.g. Company Google, Corporate SAML"
              disabled={!isEnabled}
            />
            {errors.displayName && <FieldError>{errors.displayName}</FieldError>}
          </Field>

          <div className="space-y-4">
            <Field orientation="horizontal">
              <Switch checked={isEnabled} onCheckedChange={setIsEnabled} />
              <FieldContent>
                <FieldTitle>Enable SSO</FieldTitle>
                <FieldDescription>
                  Allow users to authenticate via this SSO provider
                </FieldDescription>
              </FieldContent>
            </Field>
            <Field orientation="horizontal">
              <Switch checked={autoCreateUser} onCheckedChange={setAutoCreateUser} disabled={!isEnabled} />
              <FieldContent>
                <FieldTitle>Auto-create users</FieldTitle>
                <FieldDescription>
                  Automatically create new user accounts on first SSO login (requires allowed domains)
                </FieldDescription>
              </FieldContent>
            </Field>
          </div>

          <Field data-invalid={!!errors.allowedDomains}>
            <FieldLabel>Allowed Domains (comma-separated)</FieldLabel>
            <Input
              value={allowedDomains}
              onChange={(e) => { setAllowedDomains(e.target.value); validate() }}
              placeholder="example.com, corp.example.com"
              disabled={!isEnabled}
            />
            {errors.allowedDomains && <FieldError>{errors.allowedDomains}</FieldError>}
            {autoCreateUser && !errors.allowedDomains && (
              <p className="text-xs text-muted-foreground">
                Required when auto-create is enabled. Only users from these domains can be auto-created.
              </p>
            )}
          </Field>

          {provider === "saml" ? (
            <SAMLConfigForm
              samlEntityId={samlEntityId}
              samlSsoUrl={samlSsoUrl}
              samlCertificate={samlCertificate}
              samlAttrEmail={samlAttrEmail}
              samlAttrFirstName={samlAttrFirstName}
              samlAttrLastName={samlAttrLastName}
              onSamlEntityIdChange={setSamlEntityId}
              onSamlSsoUrlChange={setSamlSsoUrl}
              onSamlCertificateChange={setSamlCertificate}
              onSamlAttrEmailChange={setSamlAttrEmail}
              onSamlAttrFirstNameChange={setSamlAttrFirstName}
              onSamlAttrLastNameChange={setSamlAttrLastName}
              errors={errors}
              disabled={!isEnabled}
            />
          ) : (
            <OAuthConfigForm
              provider={provider}
              oauthClientId={oauthClientId}
              oauthClientSecret={oauthClientSecret}
              googleHostedDomain={googleHostedDomain}
              githubOrgs={githubOrgs}
              onOauthClientIdChange={setOauthClientId}
              onOauthClientSecretChange={setOauthClientSecret}
              onGoogleHostedDomainChange={setGoogleHostedDomain}
              onGithubOrgsChange={setGithubOrgs}
              errors={errors}
              disabled={!isEnabled}
              isEditing={isEditing}
            />
          )}

          <div className="flex gap-2 justify-end">
            <Button variant="outline" type="button" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !isEnabled}>
              {isPending ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Save className="h-4 w-4 mr-1" />
              )}
              {isEditing ? "Update" : "Create"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
    </>
  )
}
