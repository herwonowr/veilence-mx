"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { Badge } from "@/components/ui/badge"
import { Save, RefreshCw, Play, RotateCcw } from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { settingsSchema } from "@/lib/validations"
import { ZodError } from "zod"
import { useSettings, useUpdateSettings, useReanalyzeAll, useQueueStats, useRetryDeadJobs } from "@/features/settings"

export default function SettingsPage() {
  return (
    <ProtectedRoute>
      <SettingsContent />
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

  // React-recommended "store previous props" pattern for syncing derived state
  if (settingsRes?.data) {
    const key = JSON.stringify(settingsRes.data)
    if (prevSettingsKey !== key) {
      setPrevSettingsKey(key)
      setLocalSettings(settingsRes.data)
    }
  }

  const updateSetting = (key: string, value: string) => {
    setLocalSettings((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = () => {
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
  }

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
            <div>
              <label htmlFor="pypi-interval" className="text-sm font-medium">
                PyPI Poll Interval
              </label>
              <Input
                id="pypi-interval"
                value={localSettings.pypi_poll_interval ?? ""}
                onChange={(e) =>
                  updateSetting("pypi_poll_interval", e.target.value)
                }
                placeholder="5m"
              />
              <p className="text-xs text-muted-foreground mt-1">
                Go duration format (e.g., 5m, 1h, 30s)
              </p>
            </div>
            <div>
              <label htmlFor="npm-interval" className="text-sm font-medium">
                npm Poll Interval
              </label>
              <Input
                id="npm-interval"
                value={localSettings.npm_poll_interval ?? ""}
                onChange={(e) =>
                  updateSetting("npm_poll_interval", e.target.value)
                }
                placeholder="5m"
              />
              <p className="text-xs text-muted-foreground mt-1">
                Go duration format (e.g., 5m, 1h, 30s)
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Top-N Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="pypi-top-n" className="text-sm font-medium">
                PyPI Top N
              </label>
              <Input
                id="pypi-top-n"
                type="number"
                value={localSettings.pypi_top_n ?? ""}
                onChange={(e) => updateSetting("pypi_top_n", e.target.value)}
                placeholder="100"
              />
            </div>
            <div>
              <label htmlFor="npm-top-n" className="text-sm font-medium">
                npm Top N
              </label>
              <Input
                id="npm-top-n"
                type="number"
                value={localSettings.npm_top_n ?? ""}
                onChange={(e) => updateSetting("npm_top_n", e.target.value)}
                placeholder="100"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Version Analysis Depth</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            How many versions below latest to analyze when a package update is detected.
          </p>
          <div className="flex flex-col gap-3">
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
          {localSettings.version_depth_mode === "custom" && (
            <div className="ml-6 max-w-xs">
              <label htmlFor="depth-count" className="text-sm font-medium">
                Number of versions (1–5)
              </label>
              <Input
                id="depth-count"
                type="number"
                min={1}
                max={5}
                value={localSettings.version_depth_count ?? ""}
                onChange={(e) =>
                  updateSetting("version_depth_count", e.target.value)
                }
                placeholder="3"
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
          <div>
            <label htmlFor="diff-size-limit" className="text-sm font-medium">
              Diff Size Limit (bytes)
            </label>
            <Input
              id="diff-size-limit"
              type="number"
              value={localSettings.diff_size_limit ?? ""}
              onChange={(e) =>
                updateSetting("diff_size_limit", e.target.value)
              }
              placeholder="102400"
            />
            <p className="text-xs text-muted-foreground mt-1">
              Maximum diff size sent to LLM for analysis. Default: 100KB.
            </p>
          </div>
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

      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-4">
          <Button onClick={handleSave} disabled={updateMutation.isPending}>
            <Save className="h-4 w-4 mr-2" />
            {updateMutation.isPending ? "Saving..." : "Save Settings"}
          </Button>
          {updateMutation.isSuccess && (
            <span className="text-sm text-green-600">Settings saved successfully.</span>
          )}
        </div>
        {Object.keys(validationErrors).length > 0 && (
          <div className="text-sm text-destructive" role="alert">
            {Object.values(validationErrors).map((msg) => (
              <p key={msg}>{msg}</p>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
