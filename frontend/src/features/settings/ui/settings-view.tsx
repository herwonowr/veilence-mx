"use client"

import { useState, useEffect, useMemo, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle, Button, Input, Field, FieldLabel, FieldDescription, FieldError, Checkbox, Label, RadioGroup, RadioGroupItem, Alert, AlertDescription } from "@/ui"
import { Save, RefreshCw, Mail, AlertCircle, Radar, Activity, Info, AlertTriangle, Loader2 } from "lucide-react"
import { settingsSchema } from "@/domains/settings"
import { ZodError } from "zod"
import { useSettings, useUpdateSettings, useDiscoverNow, usePackageCountSummary } from "@/features/settings/hooks/use-settings"
import { useCurrentWorkspaceRole, hasMinimumRole } from "@/core"
import Link from "next/link"

export const SettingsView = () => {
  const { role: currentRole } = useCurrentWorkspaceRole()
  const canEdit = hasMinimumRole(currentRole, "admin")
  const [localSettings, setLocalSettings] = useState<Record<string, string>>({})
  const [prevSettingsKey, setPrevSettingsKey] = useState<string | null>(null)
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({})

  const { data: settingsRes } = useSettings()
  const updateMutation = useUpdateSettings()
  const discoverMutation = useDiscoverNow()
  const { data: packageSummary } = usePackageCountSummary()

  const serverSettings = settingsRes?.data ?? null

  // React-recommended "store previous props" pattern for syncing derived state
  if (settingsRes?.data) {
    const key = JSON.stringify(settingsRes.data)
    if (prevSettingsKey !== key) {
      setPrevSettingsKey(key)
      setLocalSettings(settingsRes.data)
    }
  }

  const isDirty = useMemo(() => {
    if (!serverSettings) return false
    return Object.keys(localSettings).some(
      (key) => localSettings[key] !== serverSettings[key]
    )
  }, [localSettings, serverSettings])

  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = ""
    }
    window.addEventListener("beforeunload", handler)
    return () => window.removeEventListener("beforeunload", handler)
  }, [isDirty])

  const updateSetting = (key: string, value: string) => {
    setLocalSettings((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = useCallback(() => {
    setValidationErrors({})
    try {
      settingsSchema.parse(localSettings)
      updateMutation.mutate(localSettings)
    } catch (err) {
      if (err instanceof ZodError) {
        const fieldErrors: Record<string, string> = {}
        for (const issue of err.issues) {
          const key = issue.path[0]
          if (typeof key === "string") fieldErrors[key] = issue.message
        }
        setValidationErrors(fieldErrors)
      }
    }
  }, [localSettings, updateMutation])

  if (!canEdit) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Settings</h1>
          <p className="mt-1 text-muted-foreground">
            You do not have permission to view or edit settings. Admin access is required.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="mt-1 text-muted-foreground">
          Configure monitoring, discovery, and notifications.
        </p>
      </div>

      <div className="space-y-8">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="size-5" />
            Monitoring
          </CardTitle>
          <CardDescription>
            How often to check all monitored packages for new releases.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
            <Field data-invalid={!!validationErrors.monitoring_interval}>
              <FieldLabel htmlFor="monitoring-interval">Monitoring Interval</FieldLabel>
              <Input
                id="monitoring-interval"
                value={localSettings.monitoring_interval ?? ""}
                onChange={(e) =>
                  updateSetting("monitoring_interval", e.target.value)
                }
                placeholder="1h"
              />
              {validationErrors.monitoring_interval && (
                <FieldError>{validationErrors.monitoring_interval}</FieldError>
              )}
              <FieldDescription>Duration format (e.g., 30m, 1h, 6h)</FieldDescription>
            </Field>
          </div>
        </CardContent>
      </Card>

      {/* Discovery */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Radar className="size-5" />
                Discovery
              </CardTitle>
              <CardDescription className="mt-1.5">
                Automatically scan registry popularity rankings and add new packages to monitoring.
              </CardDescription>
            </div>
            {canEdit && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => discoverMutation.mutate(undefined)}
              disabled={discoverMutation.isPending}
            >
              {discoverMutation.isPending ? (
                <Loader2 className="mr-2 size-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 size-4" />
              )}
              Discover Now
            </Button>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
            <Field data-invalid={!!validationErrors.discovery_scan_depth}>
              <FieldLabel htmlFor="discovery-scan-depth">Discovery Scan Depth</FieldLabel>
              <Input
                id="discovery-scan-depth"
                type="number"
                min={1}
                value={localSettings.discovery_scan_depth ?? ""}
                onChange={(e) => updateSetting("discovery_scan_depth", e.target.value)}
                placeholder="50"
              />
              {validationErrors.discovery_scan_depth && (
                <FieldError>{validationErrors.discovery_scan_depth}</FieldError>
              )}
              <FieldDescription>Top N packages per ecosystem per cycle</FieldDescription>
            </Field>
            <Field data-invalid={!!validationErrors.discovery_interval}>
              <FieldLabel htmlFor="discovery-interval">Discovery Interval</FieldLabel>
              <Input
                id="discovery-interval"
                value={localSettings.discovery_interval ?? ""}
                onChange={(e) =>
                  updateSetting("discovery_interval", e.target.value)
                }
                placeholder="24h"
              />
              {validationErrors.discovery_interval && (
                <FieldError>{validationErrors.discovery_interval}</FieldError>
              )}
              <FieldDescription>Duration format (e.g., 12h, 24h, 7d)</FieldDescription>
            </Field>
          </div>

          <div className="flex items-start gap-2 rounded-md bg-muted/50 p-3 text-sm text-muted-foreground">
            <Info className="mt-0.5 size-4 shrink-0" />
            <span>
              The total number of monitored packages grows over time as new packages enter the popularity rankings.
              Discovery only adds packages - it never removes them.
            </span>
          </div>

          {packageSummary && (
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span>
                Currently monitoring <strong>{packageSummary.activeCount}</strong> package{packageSummary.activeCount !== 1 ? "s" : ""}.
              </span>
              {packageSummary.suggestionsCount > 0 ? (
                <Link href="/packages/suggestions" className="text-primary hover:underline">
                  <strong>{packageSummary.suggestionsCount}</strong> suggestion{packageSummary.suggestionsCount !== 1 ? "s" : ""} pending review.
                </Link>
              ) : (
                <span className="text-muted-foreground">No pending suggestions.</span>
              )}
            </div>
          )}

          {(() => {
            const threshold = parseInt(localSettings.package_count_warning_threshold ?? "0", 10)
            const count = packageSummary?.activeCount ?? 0
            return threshold > 0 && count > threshold ? (
              <Alert variant="warning">
                <AlertTriangle className="size-4" />
                <AlertDescription>
                  You are monitoring <strong>{count}</strong> packages, which exceeds your warning threshold of <strong>{threshold}</strong>.
                </AlertDescription>
              </Alert>
            ) : null
          })()}

          <Field orientation="horizontal" data-invalid={!!validationErrors.discovery_auto_approve}>
            <Checkbox
              id="auto-approve"
              checked={localSettings.discovery_auto_approve === "true"}
              onCheckedChange={(checked) =>
                updateSetting("discovery_auto_approve", String(checked))
              }
            />
            <FieldLabel htmlFor="auto-approve" className="cursor-pointer">
              Auto-approve discovered packages
            </FieldLabel>
          </Field>
          {validationErrors.discovery_auto_approve && (
            <FieldError>{validationErrors.discovery_auto_approve}</FieldError>
          )}
          {localSettings.discovery_auto_approve === "true" && (
            <Alert variant="warning">
              <AlertTriangle className="size-4" />
              <AlertDescription>
                Discovered packages will be automatically added to active monitoring without manual review.
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
            <Field data-invalid={!!validationErrors.stale_auto_remove_months}>
              <FieldLabel htmlFor="stale-auto-remove-months">Auto-remove stale packages after (months)</FieldLabel>
              <Input
                id="stale-auto-remove-months"
                type="number"
                value={localSettings.stale_auto_remove_months ?? ""}
                onChange={(e) => updateSetting("stale_auto_remove_months", e.target.value)}
                placeholder="0"
              />
              {validationErrors.stale_auto_remove_months && (
                <FieldError>{validationErrors.stale_auto_remove_months}</FieldError>
              )}
              <FieldDescription>Packages with no updates in this many months will be automatically removed. Set to 0 to disable.</FieldDescription>
            </Field>
            <Field data-invalid={!!validationErrors.package_count_warning_threshold}>
              <FieldLabel htmlFor="package-count-warning-threshold">Package count warning threshold</FieldLabel>
              <Input
                id="package-count-warning-threshold"
                type="number"
                value={localSettings.package_count_warning_threshold ?? ""}
                onChange={(e) => updateSetting("package_count_warning_threshold", e.target.value)}
                placeholder="0"
              />
              {validationErrors.package_count_warning_threshold && (
                <FieldError>{validationErrors.package_count_warning_threshold}</FieldError>
              )}
              <FieldDescription>Show a warning when monitored packages exceed this count. Set to 0 to disable.</FieldDescription>
            </Field>
          </div>
        </CardContent>
      </Card>

      {/* Email Digest */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Mail className="size-5" />
            Email Digest
          </CardTitle>
          <CardDescription>
            Receive periodic email summaries of new alerts, analysis results, and classification breakdowns.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <Field orientation="horizontal" data-invalid={!!validationErrors.email_digest_enabled}>
            <Checkbox
              id="digest-enabled"
              checked={localSettings.email_digest_enabled === "true"}
              onCheckedChange={(checked) =>
                updateSetting("email_digest_enabled", String(checked))
              }
            />
            <FieldLabel htmlFor="digest-enabled" className="cursor-pointer">
              Enable email digest
            </FieldLabel>
          </Field>
          {validationErrors.email_digest_enabled && (
            <FieldError>{validationErrors.email_digest_enabled}</FieldError>
          )}
          {localSettings.email_digest_enabled === "true" && (
            <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
              <Field data-invalid={!!validationErrors.email_digest_frequency}>
                <FieldLabel id="digest-frequency-label">Frequency</FieldLabel>
                <RadioGroup
                  value={localSettings.email_digest_frequency ?? "daily"}
                  onValueChange={(v) => updateSetting("email_digest_frequency", v)}
                  aria-labelledby="digest-frequency-label"
                  className="mt-1 flex flex-col gap-2"
                >
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="daily" id="digest-daily" />
                    <Label htmlFor="digest-daily" className="cursor-pointer text-sm font-normal">Daily</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="weekly" id="digest-weekly" />
                    <Label htmlFor="digest-weekly" className="cursor-pointer text-sm font-normal">Weekly</Label>
                  </div>
                </RadioGroup>
                {validationErrors.email_digest_frequency && (
                  <FieldError>{validationErrors.email_digest_frequency}</FieldError>
                )}
              </Field>
              <Field data-invalid={!!validationErrors.email_digest_recipients}>
                <FieldLabel htmlFor="digest-recipients">Recipients</FieldLabel>
                <Input
                  id="digest-recipients"
                  value={localSettings.email_digest_recipients ?? ""}
                  onChange={(e) =>
                    updateSetting("email_digest_recipients", e.target.value)
                  }
                  placeholder="admin@example.com, ops@example.com"
                />
                {validationErrors.email_digest_recipients && (
                  <FieldError>{validationErrors.email_digest_recipients}</FieldError>
                )}
                <FieldDescription>Comma-separated email addresses</FieldDescription>
              </Field>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Save button - right-aligned */}
      {canEdit && (
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateMutation.isPending}>
          {updateMutation.isPending ? (
            <Loader2 className="mr-2 size-4 animate-spin" />
          ) : (
            <Save className="mr-2 size-4" />
          )}
          Save Settings
        </Button>
      </div>
      )}

      {/* Spacer for sticky footer */}
      {canEdit && isDirty && <div className="h-16" />}

      {/* Sticky unsaved changes footer */}
      {canEdit && isDirty && (
        <div className="fixed bottom-0 left-0 right-0 z-50 border-t bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/80 shadow-[0_-2px_10px_rgba(0,0,0,0.1)]">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-3">
            <div className="flex items-center gap-2 text-sm text-orange-600 dark:text-orange-400">
              <AlertCircle className="size-4" />
              <span className="font-medium">You have unsaved changes</span>
            </div>
            <div className="flex items-center gap-3">
              {updateMutation.isSuccess && (
                <span className="text-sm text-green-600">Saved!</span>
              )}
              <Button onClick={handleSave} disabled={updateMutation.isPending} size="sm">
                <Save className="mr-2 size-4" />
                {updateMutation.isPending ? "Saving..." : "Save Settings"}
              </Button>
            </div>
          </div>
        </div>
      )}
      </div>
    </div>
  )
}
