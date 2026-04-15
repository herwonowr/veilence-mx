"use client"

import { useState, useEffect, useMemo, useCallback } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import { Button } from "@/ui/components/button"
import { Separator } from "@/ui/components/separator"
import { Input } from "@/ui/components/input"
import { Field, FieldLabel, FieldDescription, FieldError } from "@/ui/components/field"
import { Badge } from "@/ui/components/badge"
import { Save, RefreshCw, Play, RotateCcw, Mail, AlertCircle, Radar, Activity, Info, AlertTriangle, ShieldCheck } from "lucide-react"
import { Checkbox } from "@/ui/components/checkbox"
import { Label } from "@/ui/components/label"
import { RadioGroup, RadioGroupItem } from "@/ui/components/radio-group"
import { Alert, AlertDescription } from "@/ui/components/alert"
import { settingsSchema } from "@/domains/settings"
import { ZodError } from "zod"
import { useSettings, useUpdateSettings, useReanalyzeAll, useDiscoverNow, usePackageCountSummary } from "@/features/settings/hooks/use-settings"
import { useQueueStats, useRetryDeadJobs } from "@/features/settings/hooks/use-queue"
import Link from "next/link"

export const SettingsView = () => {
  const [localSettings, setLocalSettings] = useState<Record<string, string>>({})
  const [prevSettingsKey, setPrevSettingsKey] = useState<string | null>(null)
  const [queueMessage, setQueueMessage] = useState("")
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({})

  const { data: settingsRes } = useSettings()
  const { data: queueRes, refetch: refetchQueue, isRefetching: queueRefetching } = useQueueStats()
  const updateMutation = useUpdateSettings()
  const reanalyzeMutation = useReanalyzeAll()
  const retryMutation = useRetryDeadJobs()
  const discoverMutation = useDiscoverNow()
  const { data: packageSummary } = usePackageCountSummary()

  const queueStats = queueRes?.data ?? null

  // The server-loaded settings (source of truth for dirty detection)
  const serverSettings = settingsRes?.data ?? null

  // React-recommended "store previous props" pattern for syncing derived state
  if (settingsRes?.data) {
    const key = JSON.stringify(settingsRes.data)
    if (prevSettingsKey !== key) {
      setPrevSettingsKey(key)
      setLocalSettings(settingsRes.data)
    }
  }

  // Dirty state: has anything changed from server-loaded values?
  const isDirty = useMemo(() => {
    if (!serverSettings) return false
    return Object.keys(localSettings).some(
      (key) => localSettings[key] !== serverSettings[key]
    )
  }, [localSettings, serverSettings])

  // Warn before closing/refreshing the browser tab with unsaved changes
  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      // Modern browsers show a generic message; returnValue is required for legacy
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

  const handleReanalyze = async () => {
    setQueueMessage("")
    try {
      const { data } = await reanalyzeMutation.mutateAsync()
      setQueueMessage(
        `Queued ${data.queued} job(s)` +
          (data.dead_retried > 0 ? `, retried ${data.dead_retried} dead job(s)` : "")
      )
      refetchQueue()
    } catch (err) {
      setQueueMessage(err instanceof Error ? err.message : "Failed to trigger re-analysis")
    }
  }

  const handleRetryDead = async (type: string) => {
    setQueueMessage("")
    try {
      const { data } = await retryMutation.mutateAsync(type)
      setQueueMessage(`Retried ${data.count} dead ${type} job(s)`)
      refetchQueue()
    } catch (err) {
      setQueueMessage(err instanceof Error ? err.message : "Failed to retry dead jobs")
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Settings</h1>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Monitoring
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            How often to check all monitored packages for new releases.
          </p>
          <div className="max-w-xs">
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
              <FieldDescription>Go duration format (e.g., 30m, 1h, 6h)</FieldDescription>
            </Field>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Radar className="h-5 w-5" />
              Discovery
            </CardTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={() => discoverMutation.mutate(undefined)}
              disabled={discoverMutation.isPending}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${discoverMutation.isPending ? "animate-spin" : ""}`} />
              Discover Now
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Automatically scan registry popularity rankings and add new packages to monitoring.
            The scan depth controls how deep into each ecosystem&apos;s rankings to look each cycle
            (e.g., 50 = top 50 PyPI + top 50 NPM).
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field data-invalid={!!validationErrors.discovery_scan_depth}>
              <FieldLabel htmlFor="discovery-scan-depth">Discovery Scan Depth</FieldLabel>
              <Input
                id="discovery-scan-depth"
                type="number"
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
              <FieldDescription>Go duration format (e.g., 12h, 24h, 7d)</FieldDescription>
            </Field>
          </div>
          <div className="flex items-start gap-2 rounded-md bg-muted/50 p-3 text-sm text-muted-foreground">
            <Info className="h-4 w-4 mt-0.5 shrink-0" />
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
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  You are monitoring <strong>{count}</strong> packages, which exceeds your warning threshold of <strong>{threshold}</strong>.
                </AlertDescription>
              </Alert>
            ) : null
          })()}
          <Separator />
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
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                Discovered packages will be automatically added to active monitoring without manual review.
              </AlertDescription>
            </Alert>
          )}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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

      <Card>
        <CardHeader>
          <CardTitle>Analyzer</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <span className="text-sm font-medium">
              Analyzer Backend
            </span>
            <p className="text-sm text-muted-foreground mt-1">
              Using <strong>copilot-api</strong> proxy → GitHub Copilot → Claude Sonnet 4.6
            </p>
          </div>
          <Separator />
          <Field data-invalid={!!validationErrors.diff_size_limit}>
            <FieldLabel htmlFor="diff-size-limit">Diff Size Limit (bytes)</FieldLabel>
            <Input
              id="diff-size-limit"
              type="number"
              value={localSettings.diff_size_limit ?? ""}
              onChange={(e) =>
                updateSetting("diff_size_limit", e.target.value)
              }
              placeholder="102400"
            />
            {validationErrors.diff_size_limit && (
              <FieldError>{validationErrors.diff_size_limit}</FieldError>
            )}
            <FieldDescription>Maximum diff size sent to LLM for analysis. Default: 100KB.</FieldDescription>
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" />
            Security
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Authentication and access control settings.
          </p>
          <Field orientation="horizontal" data-invalid={!!validationErrors.require_email_verification}>
            <Checkbox
              id="require-email-verification"
              checked={localSettings.require_email_verification === "true"}
              onCheckedChange={(checked) =>
                updateSetting("require_email_verification", String(checked))
              }
            />
            <FieldLabel htmlFor="require-email-verification" className="cursor-pointer">
              Require Email Verification
            </FieldLabel>
          </Field>
          <p className="text-sm text-muted-foreground ml-6">
            When enabled, users must verify their email address before they can log in.
          </p>
          {validationErrors.require_email_verification && (
            <FieldError>{validationErrors.require_email_verification}</FieldError>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Mail className="h-5 w-5" />
            Email Digest
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Receive periodic email summaries of new alerts, analysis results, and classification breakdowns.
          </p>
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
            <div className="space-y-4 ml-6">
              <Field data-invalid={!!validationErrors.email_digest_frequency}>
                <FieldLabel id="digest-frequency-label">Frequency</FieldLabel>
                <RadioGroup
                  value={localSettings.email_digest_frequency ?? "daily"}
                  onValueChange={(v) => updateSetting("email_digest_frequency", v)}
                  aria-labelledby="digest-frequency-label"
                  className="flex flex-col gap-2 mt-1"
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

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Analysis Queue</CardTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetchQueue()}
              disabled={queueRefetching}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${queueRefetching ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {queueStats ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <h4 className="text-sm font-medium">Diff Queue</h4>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="secondary">{queueStats.diff.pending} pending</Badge>
                  <Badge variant="default">{queueStats.diff.processing} processing</Badge>
                  <Badge variant="outline">{queueStats.diff.completed} completed</Badge>
                  {queueStats.diff.failed > 0 && (
                    <Badge variant="secondary">{queueStats.diff.failed} failed</Badge>
                  )}
                  {queueStats.diff.dead > 0 && (
                    <Badge variant="destructive">{queueStats.diff.dead} dead</Badge>
                  )}
                </div>
                {queueStats.diff.dead > 0 && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleRetryDead("diff")}
                    disabled={retryMutation.isPending}
                  >
                    <RotateCcw className="h-3 w-3 mr-1" />
                    Retry dead diff jobs
                  </Button>
                )}
              </div>
              <div className="space-y-2">
                <h4 className="text-sm font-medium">Analyze Queue</h4>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="secondary">{queueStats.analyze.pending} pending</Badge>
                  <Badge variant="default">{queueStats.analyze.processing} processing</Badge>
                  <Badge variant="outline">{queueStats.analyze.completed} completed</Badge>
                  {queueStats.analyze.failed > 0 && (
                    <Badge variant="secondary">{queueStats.analyze.failed} failed</Badge>
                  )}
                  {queueStats.analyze.dead > 0 && (
                    <Badge variant="destructive">{queueStats.analyze.dead} dead</Badge>
                  )}
                </div>
                {queueStats.analyze.dead > 0 && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleRetryDead("analyze")}
                    disabled={retryMutation.isPending}
                  >
                    <RotateCcw className="h-3 w-3 mr-1" />
                    Retry dead analyze jobs
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              Queue stats unavailable. Redis may not be running.
            </p>
          )}
          <Separator />
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              onClick={handleReanalyze}
              disabled={reanalyzeMutation.isPending}
            >
              <Play className="h-4 w-4 mr-2" />
              {reanalyzeMutation.isPending ? "Queuing..." : "Re-analyze Unanalyzed Diffs"}
            </Button>
            {queueMessage && (
              <span className="text-sm text-muted-foreground">{queueMessage}</span>
            )}
          </div>
        </CardContent>
      </Card>

      <div className="flex items-center gap-4">
        <Button onClick={handleSave} disabled={updateMutation.isPending}>
          <Save className="h-4 w-4 mr-2" />
          {updateMutation.isPending ? "Saving..." : "Save Settings"}
        </Button>
        {updateMutation.isSuccess && (
          <span className="text-sm text-green-600">Settings saved successfully.</span>
        )}
      </div>

      {/* Spacer for sticky footer */}
      {isDirty && <div className="h-16" />}

      {/* Sticky unsaved changes footer */}
      {isDirty && (
        <div className="fixed bottom-0 left-0 right-0 z-50 border-t bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 shadow-[0_-2px_10px_rgba(0,0,0,0.1)]">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-3">
            <div className="flex items-center gap-2 text-sm text-orange-600 dark:text-orange-400">
              <AlertCircle className="h-4 w-4" />
              <span className="font-medium">You have unsaved changes</span>
            </div>
            <div className="flex items-center gap-3">
              {updateMutation.isSuccess && (
                <span className="text-sm text-green-600">Saved!</span>
              )}
              <Button onClick={handleSave} disabled={updateMutation.isPending} size="sm">
                <Save className="h-4 w-4 mr-2" />
                {updateMutation.isPending ? "Saving..." : "Save Settings"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
