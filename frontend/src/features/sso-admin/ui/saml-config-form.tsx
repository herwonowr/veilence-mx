"use client"

import { useState } from "react"
import { Input, Field, FieldLabel, FieldError, Textarea, Button } from "@/ui"
import { Link, Loader2 } from "lucide-react"
import { useImportSAMLMetadata } from "@/features/sso-admin/hooks/use-sso-configs"

interface SAMLConfigFormProps {
  samlEntityId: string
  samlSsoUrl: string
  samlCertificate: string
  samlAttrEmail: string
  samlAttrFirstName: string
  samlAttrLastName: string
  onSamlEntityIdChange: (v: string) => void
  onSamlSsoUrlChange: (v: string) => void
  onSamlCertificateChange: (v: string) => void
  onSamlAttrEmailChange: (v: string) => void
  onSamlAttrFirstNameChange: (v: string) => void
  onSamlAttrLastNameChange: (v: string) => void
  errors?: Record<string, string>
  disabled?: boolean
}

export const SAMLConfigForm = ({
  samlEntityId,
  samlSsoUrl,
  samlCertificate,
  samlAttrEmail,
  samlAttrFirstName,
  samlAttrLastName,
  onSamlEntityIdChange,
  onSamlSsoUrlChange,
  onSamlCertificateChange,
  onSamlAttrEmailChange,
  onSamlAttrFirstNameChange,
  onSamlAttrLastNameChange,
  errors = {},
  disabled = false,
}: SAMLConfigFormProps) => {
  const [showImport, setShowImport] = useState(false)
  const [metadataUrl, setMetadataUrl] = useState("")
  const importMutation = useImportSAMLMetadata()

  const handleImport = () => {
    if (!metadataUrl.trim()) return
    importMutation.mutate(metadataUrl.trim(), {
      onSuccess: (response) => {
        const data = response.data
        if (data) {
          onSamlEntityIdChange(data.entityId)
          onSamlSsoUrlChange(data.ssoUrl)
          onSamlCertificateChange(data.certificate)
        }
        setShowImport(false)
        setMetadataUrl("")
      },
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="font-medium text-sm">SAML Settings</h4>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => setShowImport(!showImport)}
          disabled={disabled}
        >
          <Link className="h-4 w-4 mr-1" />
          Import from URL
        </Button>
      </div>

      {showImport && (
        <div className="flex gap-2 items-start rounded-md border p-3 bg-muted/50">
          <Input
            value={metadataUrl}
            onChange={(e) => setMetadataUrl(e.target.value)}
            placeholder="https://your-idp.example.com/saml/metadata"
            disabled={disabled || importMutation.isPending}
            className="flex-1"
          />
          <Button
            type="button"
            size="sm"
            onClick={handleImport}
            disabled={disabled || importMutation.isPending || !metadataUrl.trim()}
          >
            {importMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              "Import"
            )}
          </Button>
        </div>
      )}

      <Field data-invalid={!!errors.samlEntityId}>
        <FieldLabel>IdP Entity ID</FieldLabel>
        <Input
          value={samlEntityId}
          onChange={(e) => onSamlEntityIdChange(e.target.value)}
          placeholder="https://your-idp.yourcompany.com/saml/metadata"
          disabled={disabled}
        />
        {errors.samlEntityId && <FieldError>{errors.samlEntityId}</FieldError>}
      </Field>

      <Field data-invalid={!!errors.samlSsoUrl}>
        <FieldLabel>SSO URL</FieldLabel>
        <Input
          value={samlSsoUrl}
          onChange={(e) => onSamlSsoUrlChange(e.target.value)}
          placeholder="https://your-idp.yourcompany.com/saml/sso"
          disabled={disabled}
        />
        {errors.samlSsoUrl && <FieldError>{errors.samlSsoUrl}</FieldError>}
      </Field>

      <Field data-invalid={!!errors.samlCertificate}>
        <FieldLabel>IdP Certificate (PEM)</FieldLabel>
        <Textarea
          value={samlCertificate}
          onChange={(e) => onSamlCertificateChange(e.target.value)}
          placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
          rows={5}
          disabled={disabled}
          className="font-mono"
        />
        {errors.samlCertificate && <FieldError>{errors.samlCertificate}</FieldError>}
      </Field>

      <h4 className="font-medium text-sm">Attribute Mapping</h4>

      <div className="grid grid-cols-3 gap-4">
        <Field data-invalid={!!errors.samlAttrEmail}>
          <FieldLabel>Email Attribute</FieldLabel>
          <Input
            value={samlAttrEmail}
            onChange={(e) => onSamlAttrEmailChange(e.target.value)}
            disabled={disabled}
          />
          {errors.samlAttrEmail && <FieldError>{errors.samlAttrEmail}</FieldError>}
        </Field>
        <Field data-invalid={!!errors.samlAttrFirstName}>
          <FieldLabel>First Name Attribute</FieldLabel>
          <Input
            value={samlAttrFirstName}
            onChange={(e) => onSamlAttrFirstNameChange(e.target.value)}
            disabled={disabled}
          />
          {errors.samlAttrFirstName && <FieldError>{errors.samlAttrFirstName}</FieldError>}
        </Field>
        <Field data-invalid={!!errors.samlAttrLastName}>
          <FieldLabel>Last Name Attribute</FieldLabel>
          <Input
            value={samlAttrLastName}
            onChange={(e) => onSamlAttrLastNameChange(e.target.value)}
            disabled={disabled}
          />
          {errors.samlAttrLastName && <FieldError>{errors.samlAttrLastName}</FieldError>}
        </Field>
      </div>
    </div>
  )
}
