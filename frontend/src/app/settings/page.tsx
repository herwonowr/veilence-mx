"use client"

import { useState, useEffect, useMemo, useCallback } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { FormField } from "@/components/form-field"
import { Badge } from "@/components/ui/badge"
import { Save, RefreshCw, Play, RotateCcw, Mail, AlertCircle } from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { RequireOrg } from "@/components/require-org"
import { settingsSchema } from "@/lib/validations"
import { ZodError } from "zod"
import { useSettings, useUpdateSettings, useReanalyzeAll, useQueueStats, useRetryDeadJobs } from "@/features/settings"

export default function SettingsPage() {
  return (
    <ProtectedRoute>
      <RequireOrg feature="settings">
        <SettingsContent />
      </RequireOrg>
    </ProtectedRoute>
  )
}

function SettingsContent() {
  const [localSettings, setLocalSettings] = useState<Record<string, string>>({})
  const [prevSettingsKey, setPrevSettingsKey] = useState<string | null>(null)
  const [queueMessage, setQueueMessage] = useState("")
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({})

  const { data: settingsRes } = useSettings()
  const { data: queueRes, refetch: refetchQueue, isRefetching: queueRefetching } = useQueueStats()
  const updateMutation = useUpdateSettings()
  const reanalyzeMutation = useReanalyzeAll()
  const retryMutation = useRetryDeadJobs()

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
          <CardTitle>Poll Intervals</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <FormField
              id="pypi-interval"
              label="PyPI Poll Interval"
              value={localSettings.pypi_poll_interval ?? ""}
              onChange={(e) =>
                updateSetting("pypi_poll_interval", e.target.value)
              }
              placeholder="5m"
              error={validationErrors.pypi_poll_interval}
              description="Go duration format (e.g., 5m, 1h, 30s)"
            />
            <FormField
              id="npm-interval"
              label="npm Poll Interval"
              value={localSettings.npm_poll_interval ?? ""}
              onChange={(e) =>
                updateSetting("npm_poll_interval", e.target.value)
              }
              placeholder="5m"
              error={validationErrors.npm_poll_interval}
              description="Go duration format (e.g., 5m, 1h, 30s)"
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Top-N Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <FormField
              id="pypi-top-n"
              label="PyPI Top N"
              type="number"
              value={localSettings.pypi_top_n ?? ""}
              onChange={(e) => updateSetting("pypi_top_n", e.target.value)}
              placeholder="100"
              error={validationErrors.pypi_top_n}
            />
            <FormField
              id="npm-top-n"
              label="npm Top N"
              type="number"
              value={localSettings.npm_top_n ?? ""}
              onChange={(e) => updateSetting("npm_top_n", e.target.value)}
              placeholder="100"
              error={validationErrors.npm_top_n}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle id="version-depth-mode-label">Version Analysis Depth</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            How many versions below latest to analyze when a package update is detected.
          </p>
          <div
            className="flex flex-col gap-3"
            role="radiogroup"
            aria-labelledby="version-depth-mode-label"
            aria-describedby={validationErrors.version_depth_mode ? "version-depth-mode-error" : undefined}
            aria-invalid={!!validationErrors.version_depth_mode}
          >
            <label htmlFor="depth-latest" className="flex items-center gap-2 cursor-pointer">
              <input
                id="depth-latest"
                type="radio"
                name="version_depth_mode"
                value="latest"
                checked={(localSettings.version_depth_mode ?? "latest") === "latest"}
                onChange={() => updateSetting("version_depth_mode", "latest")}
                className="accent-primary"
              />
              <span className="text-sm font-medium">Latest only</span>
              <span className="text-xs text-muted-foreground">— analyze only the newest version</span>
            </label>
            <label htmlFor="depth-custom" className="flex items-center gap-2 cursor-pointer">
              <input
                id="depth-custom"
                type="radio"
                name="version_depth_mode"
                value="custom"
                checked={localSettings.version_depth_mode === "custom"}
                onChange={() => updateSetting("version_depth_mode", "custom")}
                className="accent-primary"
              />
              <span className="text-sm font-medium">Custom</span>
              <span className="text-xs text-muted-foreground">— analyze up to N versions below latest</span>
            </label>
          </div>
          {validationErrors.version_depth_mode && (
            <p id="version-depth-mode-error" className="text-sm text-destructive" role="alert">
              {validationErrors.version_depth_mode}
            </p>
          )}
          {localSettings.version_depth_mode === "custom" && (
            <div className="ml-6 max-w-xs">
              <FormField
                id="depth-count"
                label="Number of versions (1–5)"
                type="number"
                min={1}
                max={5}
                value={localSettings.version_depth_count ?? ""}
                onChange={(e) =>
                  updateSetting("version_depth_count", e.target.value)
                }
                placeholder="3"
                error={validationErrors.version_depth_count}
              />
            </div>
          )}
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
          <FormField
            id="diff-size-limit"
            label="Diff Size Limit (bytes)"
            type="number"
            value={localSettings.diff_size_limit ?? ""}
            onChange={(e) =>
              updateSetting("diff_size_limit", e.target.value)
            }
            placeholder="102400"
            error={validationErrors.diff_size_limit}
            description="Maximum diff size sent to LLM for analysis. Default: 100KB."
          />
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
          <div className="flex items-center gap-3">
            <label htmlFor="digest-enabled" className="flex items-center gap-2 cursor-pointer">
              <input
                id="digest-enabled"
                type="checkbox"
                checked={localSettings.email_digest_enabled === "true"}
                onChange={(e) =>
                  updateSetting("email_digest_enabled", e.target.checked ? "true" : "false")
                }
                className="accent-primary h-4 w-4"
                aria-describedby={validationErrors.email_digest_enabled ? "digest-enabled-error" : undefined}
                aria-invalid={!!validationErrors.email_digest_enabled}
              />
              <span className="text-sm font-medium">Enable email digest</span>
            </label>
          </div>
          {validationErrors.email_digest_enabled && (
            <p id="digest-enabled-error" className="text-sm text-destructive" role="alert">
              {validationErrors.email_digest_enabled}
            </p>
          )}
          {localSettings.email_digest_enabled === "true" && (
            <div className="space-y-4 ml-6">
              <div>
                <span className="text-sm font-medium" id="digest-frequency-label">Frequency</span>
                <div
                  className="flex flex-col gap-2 mt-1"
                  role="radiogroup"
                  aria-labelledby="digest-frequency-label"
                  aria-describedby={validationErrors.email_digest_frequency ? "digest-frequency-error" : undefined}
                  aria-invalid={!!validationErrors.email_digest_frequency}
                >
                  <label htmlFor="digest-daily" className="flex items-center gap-2 cursor-pointer">
                    <input
                      id="digest-daily"
                      type="radio"
                      name="email_digest_frequency"
                      value="daily"
                      checked={(localSettings.email_digest_frequency ?? "daily") === "daily"}
                      onChange={() => updateSetting("email_digest_frequency", "daily")}
                      className="accent-primary"
                    />
                    <span className="text-sm">Daily</span>
                  </label>
                  <label htmlFor="digest-weekly" className="flex items-center gap-2 cursor-pointer">
                    <input
                      id="digest-weekly"
                      type="radio"
                      name="email_digest_frequency"
                      value="weekly"
                      checked={localSettings.email_digest_frequency === "weekly"}
                      onChange={() => updateSetting("email_digest_frequency", "weekly")}
                      className="accent-primary"
                    />
                    <span className="text-sm">Weekly</span>
                  </label>
                </div>
                {validationErrors.email_digest_frequency && (
                  <p id="digest-frequency-error" className="text-sm text-destructive mt-1" role="alert">
                    {validationErrors.email_digest_frequency}
                  </p>
                )}
              </div>
              <FormField
                id="digest-recipients"
                label="Recipients"
                value={localSettings.email_digest_recipients ?? ""}
                onChange={(e) =>
                  updateSetting("email_digest_recipients", e.target.value)
                }
                placeholder="admin@example.com, ops@example.com"
                error={validationErrors.email_digest_recipients}
                description="Comma-separated email addresses"
              />
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
