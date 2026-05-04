"use client"

import { useState } from "react"
import { useAuthSettings } from "@/features/sso-admin/hooks/use-auth-settings"
import { usePlatformSSOConfigs } from "@/features/sso-admin/hooks/use-sso-configs"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Switch,
  Field,
  FieldContent,
  FieldTitle,
  FieldDescription,
  ConfirmDialog,
} from "@/ui"
import { Shield } from "lucide-react"

export const AuthSettingsCard = () => {
  const { settings, updateSettings, isPending } = useAuthSettings()
  const { data: ssoConfigs } = usePlatformSSOConfigs()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [pendingValue, setPendingValue] = useState<boolean | null>(null)

  const hasEnabledSSO = (ssoConfigs?.data ?? []).some((c) => c.isEnabled)
  const canDisablePassword = hasEnabledSSO

  const currentValue = settings?.passwordLoginEnabled ?? true

  const handleToggleRequest = (checked: boolean) => {
    setPendingValue(checked)
    setConfirmOpen(true)
  }

  const handleConfirm = () => {
    if (pendingValue !== null) {
      updateSettings({ passwordLoginEnabled: pendingValue })
    }
    setPendingValue(null)
  }

  const handleOpenChange = (open: boolean) => {
    setConfirmOpen(open)
    if (!open) {
      setPendingValue(null)
    }
  }

  return (
    <>
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={handleOpenChange}
        title={pendingValue ? "Enable password login?" : "Disable password login?"}
        description={
          pendingValue
            ? "Enabling password login will allow users to sign in with email and password in addition to SSO."
            : "Disabling password login will prevent users from signing in with email and password. Only SSO methods will be available."
        }
        actionLabel={pendingValue ? "Enable" : "Disable"}
        onConfirm={handleConfirm}
      />
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Authentication Settings
          </CardTitle>
          <CardDescription>
            Control how users authenticate to the platform.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Field orientation="horizontal">
            <Switch
              checked={currentValue}
              onCheckedChange={handleToggleRequest}
              disabled={isPending || (!canDisablePassword && currentValue)}
            />
            <FieldContent>
            <FieldTitle>Password login</FieldTitle>
            <FieldDescription>
              Allow users to sign in with email and password. Disable to enforce SSO-only login.
              {!canDisablePassword && currentValue && (
                <span className="block text-xs text-amber-600 dark:text-amber-400 mt-1">
                  Cannot disable until at least one SSO provider is enabled.
                </span>
              )}
            </FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Switch
            checked={settings?.registrationEnabled ?? true}
            disabled
          />
          <FieldContent>
            <FieldTitle>Registration</FieldTitle>
            <FieldDescription>
              Allow new users to create accounts via email/password registration.
              Controlled by <code className="text-xs">REGISTRATION_ENABLED</code> environment variable.
            </FieldDescription>
          </FieldContent>
        </Field>
      </CardContent>
      </Card>
    </>
  )
}
