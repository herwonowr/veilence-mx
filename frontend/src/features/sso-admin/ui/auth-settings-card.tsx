"use client"

import { useAuthSettings } from "@/features/sso-admin/hooks/use-auth-settings"
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
} from "@/ui"
import { Shield } from "lucide-react"

export const AuthSettingsCard = () => {
  const { settings, updateSettings, isPending } = useAuthSettings()

  return (
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
            checked={settings?.passwordLoginEnabled ?? true}
            onCheckedChange={(checked) =>
              updateSettings({ passwordLoginEnabled: checked })
            }
            disabled={isPending}
          />
          <FieldContent>
            <FieldTitle>Password login</FieldTitle>
            <FieldDescription>
              Allow users to sign in with email and password. Disable to enforce SSO-only login.
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
  )
}
