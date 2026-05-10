"use client"

import { Card, CardContent, CardDescription, CardHeader, CardTitle, Button, Badge, EmptyState, ConfirmDialog } from "@/ui"
import { Link2, Unlink, Loader2 } from "lucide-react"
import { toast } from "sonner"
import { useLinkedIdentities, useUnlinkIdentity, useSSOProviders } from "@/features/sso"
import { buildLinkIdentityUrl } from "@/domains/sso"
import { ROUTES, getStoredAccessToken, sanitizeErrorMessage } from "@/core"

const providerLabel = (provider: string) => {
  switch (provider) {
    case "local":
      return "Email/Password"
    case "saml":
      return "SAML"
    case "google":
      return "Google"
    case "github":
      return "GitHub"
    default:
      return provider
  }
}

export const LinkedIdentities = () => {
  const { data: identitiesRes, isLoading: identitiesLoading } = useLinkedIdentities()
  const unlinkMutation = useUnlinkIdentity()
  const { data: providersRes, isLoading: providersLoading } = useSSOProviders()

  const identities = identitiesRes?.data ?? []
  const providers = providersRes?.data?.providers ?? []

  // Hide entirely if no providers configured and no existing identities
  if (identitiesLoading || providersLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (providers.length === 0 && identities.length === 0) {
    return null
  }

  const linkedProviderTypes = new Set(identities.map((i) => i.provider))
  const unlinkableProviders = providers.filter((p) => !linkedProviderTypes.has(p.provider))

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2">
              <Link2 className="h-5 w-5" />
              Linked Identities
            </CardTitle>
            <CardDescription>
              Manage your linked SSO identities. Link additional providers to enable SSO login.
            </CardDescription>
          </div>
          {unlinkableProviders.length > 0 && (
            <div className="flex gap-2">
              {unlinkableProviders.map((provider) => (
                <Button
                  key={provider.id}
                  variant="outline"
                  size="sm"
                  onClick={() => { window.location.href = buildLinkIdentityUrl(provider.id, `${window.location.origin}${ROUTES.SSO_CALLBACK}?redirect=${encodeURIComponent(ROUTES.ACCOUNT)}`, getStoredAccessToken() ?? undefined) }}
                >
                  <Link2 className="h-4 w-4 mr-1" />
                  Link {provider.displayName}
                </Button>
              ))}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {identities.length === 0 ? (
          <EmptyState
            icon={<Link2 className="size-10" />}
            title="No linked identities"
            description="Link an SSO provider below."
          />
        ) : (
          <div className="space-y-2">
            {identities.map((identity) => (
              <div
                key={identity.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{providerLabel(identity.provider)}</Badge>
                    <span className="text-sm">{identity.providerEmail}</span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    Linked {new Date(identity.linkedAt).toLocaleString()}
                  </p>
                </div>
                {(
                  <ConfirmDialog
                    title="Unlink Identity"
                    description="Are you sure you want to unlink this identity? You will no longer be able to log in with this SSO provider."
                    details={[
                      { label: "Provider", value: providerLabel(identity.provider) },
                      { label: "Email", value: identity.providerEmail },
                    ]}
                    actionLabel="Unlink"
                    onConfirm={async () => {
                      try {
                        await unlinkMutation.mutateAsync(identity.id)
                        toast.success("Identity unlinked successfully.")
                      } catch (error) {
                        toast.error(sanitizeErrorMessage(error, "Failed to unlink identity"))
                      }
                    }}
                  >
                    <Button
                      variant="outline"
                      size="sm"
                    >
                      <Unlink className="h-4 w-4 mr-1" />
                      Unlink
                    </Button>
                  </ConfirmDialog>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
