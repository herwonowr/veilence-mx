"use client"

import { useState, useCallback } from "react"
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
} from "@/ui"
import { config } from "@/core"
import { useSAMLSPCertificate } from "@/features/sso-admin/hooks/use-saml-sp-certificate"
import { Copy, Check, Download } from "lucide-react"

interface SAMLSPInfoProps {
  configId?: string
}

export const SAMLSPInfo = ({ configId }: SAMLSPInfoProps) => {
  const [copiedField, setCopiedField] = useState<string | null>(null)
  const { data: certRes } = useSAMLSPCertificate()
  const spCertificate = certRes?.data?.certificate ?? ""

  const spEntityId = `${config.apiBaseUrl}/api/auth/saml/metadata`
  const acsUrl = `${config.apiBaseUrl}/api/auth/saml/acs`

  const handleCopy = useCallback((value: string, field: string) => {
    navigator.clipboard.writeText(value).then(() => {
      setCopiedField(field)
      setTimeout(() => setCopiedField(null), 2000)
    })
  }, [])

  const handleDownloadCert = useCallback(() => {
    if (!spCertificate) return
    const blob = new Blob([spCertificate], { type: "application/x-pem-file" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "veilence-mx-saml-sp.pem"
    a.click()
    URL.revokeObjectURL(url)
  }, [spCertificate])

  const fields = [
    { label: "SP Entity ID", value: spEntityId, key: "entityId" },
    { label: "ACS URL (Assertion Consumer Service)", value: acsUrl, key: "acs" },
    ...(configId ? [{ label: "SP Metadata URL", value: `${config.apiBaseUrl}/api/auth/saml/${configId}/metadata`, key: "metadata" }] : []),
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Service Provider Information</CardTitle>
        <CardDescription>
          Use these values to configure your Identity Provider (IdP).
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {fields.map(({ label, value, key }) => (
          <Field key={key}>
            <FieldLabel>{label}</FieldLabel>
            <div className="flex gap-2">
              <Input value={value} readOnly className="font-mono text-sm" />
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={() => handleCopy(value, key)}
                aria-label={`Copy ${label}`}
              >
                {copiedField === key ? (
                  <Check className="h-4 w-4 text-green-600" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </Field>
        ))}
        {spCertificate && (
          <Field>
            <FieldLabel>SP Signing Certificate</FieldLabel>
            <textarea
              value={spCertificate}
              readOnly
              rows={6}
              className="w-full rounded-md border bg-muted px-3 py-2 font-mono resize-none"
            />
            <div className="flex gap-2 mt-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => handleCopy(spCertificate, "spCert")}
              >
                {copiedField === "spCert" ? (
                  <Check className="mr-1 h-4 w-4 text-green-600" />
                ) : (
                  <Copy className="mr-1 h-4 w-4" />
                )}
                Copy
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleDownloadCert}
              >
                <Download className="mr-1 h-4 w-4" />
                Download .pem
              </Button>
            </div>
          </Field>
        )}
      </CardContent>
    </Card>
  )
}
