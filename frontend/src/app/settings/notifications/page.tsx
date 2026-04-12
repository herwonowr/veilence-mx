"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Plus,
  Trash2,
  Mail,
  Webhook,
  Loader2,
  BellRing,
  Route,
  Zap,
} from "lucide-react"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { EmptyState } from "@/components/empty-state"
import { ProtectedRoute } from "@/components/protected-route"
import { RequireOrg } from "@/components/require-org"
import { useAuth } from "@/lib/auth-context"
import {
  useChannels,
  useCreateChannel,
  useUpdateChannel,
  useDeleteChannel,
  useTestChannel,
  useRules,
  useCreateRule,
  useDeleteRule,
} from "@/features/notifications"
import type { NotificationChannel, NotificationChannelType } from "@/types"

const CHANNEL_TYPE_LABELS: Record<NotificationChannelType, string> = {
  email: "Email",
  slack: "Slack",
  webhook: "Webhook",
}

const CHANNEL_TYPE_ICONS: Record<NotificationChannelType, React.ReactNode> = {
  email: <Mail className="size-4" />,
  slack: <Webhook className="size-4" />,
  webhook: <Webhook className="size-4" />,
}

const SEVERITIES = ["low", "medium", "high", "critical"] as const

const SEVERITY_COLORS: Record<string, string> = {
  low: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  medium: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
}

export default function NotificationChannelsPage() {
  return (
    <ProtectedRoute>
      <RequireOrg feature="notification settings">
        <ChannelsContent />
      </RequireOrg>
    </ProtectedRoute>
  )
}

function ChannelsContent() {
  const { currentOrg } = useAuth()
  const orgId = currentOrg?.id ?? null

  const { data: channelsRes, isLoading: channelsLoading } = useChannels(orgId)
  const { data: rulesRes, isLoading: rulesLoading } = useRules(orgId)

  const channels = channelsRes?.data ?? []
  const rules = rulesRes?.data ?? []

  if (!orgId) {
    return (
      <div className="space-y-6">
        <h1 className="text-3xl font-bold">Notification Channels</h1>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Select an organization to manage notification channels.
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Notification Channels</h1>
        <p className="mt-1 text-muted-foreground">
          Manage how and when alerts are delivered to your team.
        </p>
      </div>

      {/* Channels Section */}
      <ChannelsSection
        orgId={orgId}
        channels={channels}
        loading={channelsLoading}
      />

      <Separator />

      {/* Rules Section */}
      <RulesSection
        orgId={orgId}
        rules={rules}
        channels={channels}
        loading={rulesLoading}
      />
    </div>
  )
}

// ─── Channels Section ──────────────────────────────────────────

function ChannelsSection({
  orgId,
  channels,
  loading,
}: {
  orgId: number
  channels: NotificationChannel[]
  loading: boolean
}) {
  const [createOpen, setCreateOpen] = useState(false)
  const [channelName, setChannelName] = useState("")
  const [channelType, setChannelType] = useState<NotificationChannelType | "">("")
  const [channelConfig, setChannelConfig] = useState("")

  const createMutation = useCreateChannel(orgId)
  const updateMutation = useUpdateChannel(orgId)
  const deleteMutation = useDeleteChannel(orgId)
  const testMutation = useTestChannel(orgId)

  const handleCreate = () => {
    if (!channelName || !channelType) return
    createMutation.mutate(
      { name: channelName, type: channelType, config: channelConfig },
      {
        onSuccess: () => {
          setCreateOpen(false)
          setChannelName("")
          setChannelType("")
          setChannelConfig("")
        },
      }
    )
  }

  const handleToggle = (channel: NotificationChannel, checked: boolean) => {
    updateMutation.mutate({
      id: channel.id,
      name: channel.name,
      config: channel.config,
      isActive: checked,
    })
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="flex items-center gap-2">
            <BellRing className="size-5" />
            Channels
          </CardTitle>
          <CardDescription>
            Configure where notifications are sent.
          </CardDescription>
        </div>
        <Dialog open={createOpen} onOpenChange={(open) => setCreateOpen(open)}>
          <DialogTrigger
            render={
              <Button size="sm">
                <Plus className="mr-1 size-4" />
                Add Channel
              </Button>
            }
          />
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Notification Channel</DialogTitle>
              <DialogDescription>
                Add a new channel to receive alert notifications.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <Label htmlFor="channel-create-name">Name</Label>
                <Input
                  id="channel-create-name"
                  placeholder="e.g., Team Slack"
                  value={channelName}
                  onChange={(e) => setChannelName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Type</Label>
                <Select
                  value={channelType}
                  onValueChange={(v) => {
                    if (v) setChannelType(v as NotificationChannelType)
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue>{channelType === "email" ? "Email (SMTP)" : channelType === "slack" ? "Slack (Webhook)" : channelType === "webhook" ? "Webhook" : "Select type"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="email">Email (SMTP)</SelectItem>
                    <SelectItem value="slack">Slack (Webhook)</SelectItem>
                    <SelectItem value="webhook">Webhook</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {channelType && (
                <ChannelConfigFields
                  type={channelType}
                  config={channelConfig}
                  onChange={setChannelConfig}
                />
              )}
            </div>
            <DialogFooter>
              <Button
                onClick={handleCreate}
                disabled={!channelName || !channelType || createMutation.isPending}
              >
                {createMutation.isPending && (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                )}
                Create Channel
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="size-6 animate-spin text-muted-foreground" />
          </div>
        ) : channels.length === 0 ? (
          <EmptyState
            icon={<BellRing className="h-8 w-8" />}
            title="No channels configured."
            description="Add a notification channel to start receiving alerts via email, Slack, or webhook."
          >
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1 size-4" />
              Add Channel
            </Button>
          </EmptyState>
        ) : (
          <div className="space-y-3">
            {channels.map((channel) => (
              <div
                key={channel.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="flex items-center gap-3">
                  <div className="flex size-8 items-center justify-center rounded-md bg-muted">
                    {CHANNEL_TYPE_ICONS[channel.type]}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{channel.name}</span>
                      <Badge variant="outline" className="text-xs">
                        {CHANNEL_TYPE_LABELS[channel.type]}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Created {new Date(channel.createdAt).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => testMutation.mutate(channel.id)}
                    disabled={testMutation.isPending}
                    aria-label={`Test channel ${channel.name}`}
                    title="Send test notification"
                  >
                    <Zap className="size-4 text-primary" />
                  </Button>
                  <Switch
                    checked={channel.isActive}
                    onCheckedChange={(checked) =>
                      handleToggle(channel, checked)
                    }
                    aria-label={`Toggle ${channel.name} ${channel.isActive ? "off" : "on"}`}
                  />
                  <ConfirmDialog
                    title="Delete Channel"
                    description={`Are you sure you want to delete "${channel.name}"? Any routing rules using this channel will also be removed.`}
                    actionLabel="Delete"
                    onConfirm={async () => { await deleteMutation.mutateAsync(channel.id) }}
                  >
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      disabled={deleteMutation.isPending}
                      aria-label={`Delete channel ${channel.name}`}
                    >
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </ConfirmDialog>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ─── Channel Config Fields ──────────────────────────────────────

function ChannelConfigFields({
  type,
  config,
  onChange,
}: {
  type: NotificationChannelType
  config: string
  onChange: (config: string) => void
}) {
  // Parse existing config
  let parsed: Record<string, string> = {}
  try {
    if (config) parsed = JSON.parse(config)
  } catch {
    /* ignore parse errors */
  }

  const updateField = (key: string, value: string) => {
    const updated = { ...parsed, [key]: value }
    onChange(JSON.stringify(updated))
  }

  switch (type) {
    case "email":
      return (
        <div className="space-y-2">
          <Label htmlFor="channel-email-host">SMTP Host</Label>
          <Input
            id="channel-email-host"
            placeholder="smtp.example.com"
            value={parsed.host ?? ""}
            onChange={(e) => updateField("host", e.target.value)}
          />
          <Label htmlFor="channel-email-port">SMTP Port</Label>
          <Input
            id="channel-email-port"
            placeholder="587"
            value={parsed.port ?? ""}
            onChange={(e) => updateField("port", e.target.value)}
          />
          <Label htmlFor="channel-email-from">From Address</Label>
          <Input
            id="channel-email-from"
            placeholder="alerts@example.com"
            value={parsed.from ?? ""}
            onChange={(e) => updateField("from", e.target.value)}
          />
          <Label htmlFor="channel-email-to">To Address</Label>
          <Input
            id="channel-email-to"
            placeholder="team@example.com"
            value={parsed.to ?? ""}
            onChange={(e) => updateField("to", e.target.value)}
          />
        </div>
      )
    case "slack":
      return (
        <div className="space-y-2">
          <Label htmlFor="channel-slack-url">Slack Webhook URL</Label>
          <Input
            id="channel-slack-url"
            placeholder="https://hooks.slack.com/services/..."
            value={parsed.webhookUrl ?? ""}
            onChange={(e) => updateField("webhookUrl", e.target.value)}
          />
        </div>
      )
    case "webhook":
      return (
        <div className="space-y-2">
          <Label htmlFor="channel-webhook-url">Webhook URL</Label>
          <Input
            id="channel-webhook-url"
            placeholder="https://api.example.com/webhook"
            value={parsed.url ?? ""}
            onChange={(e) => updateField("url", e.target.value)}
          />
          <Label htmlFor="channel-webhook-secret">Secret (optional)</Label>
          <Input
            id="channel-webhook-secret"
            type="password"
            placeholder="Signing secret for HMAC verification"
            value={parsed.secret ?? ""}
            onChange={(e) => updateField("secret", e.target.value)}
          />
        </div>
      )
    default:
      return null
  }
}

// ─── Rules Section ──────────────────────────────────────────────

function RulesSection({
  orgId,
  rules,
  channels,
  loading,
}: {
  orgId: number
  rules: Array<{ id: number; channelId: number; severity: string; isActive: boolean; createdAt: string }>
  channels: NotificationChannel[]
  loading: boolean
}) {
  const [createOpen, setCreateOpen] = useState(false)
  const [ruleChannel, setRuleChannel] = useState<number | null>(null)
  const [ruleSeverity, setRuleSeverity] = useState("")

  const createMutation = useCreateRule(orgId)
  const deleteMutation = useDeleteRule(orgId)

  const handleCreate = () => {
    if (!ruleChannel || !ruleSeverity) return
    createMutation.mutate(
      { channelId: ruleChannel, severity: ruleSeverity },
      {
        onSuccess: () => {
          setCreateOpen(false)
          setRuleChannel(null)
          setRuleSeverity("")
        },
      }
    )
  }

  const getChannelName = (channelId: number) => {
    const ch = channels.find((c) => c.id === channelId)
    return ch ? ch.name : `Channel #${channelId}`
  }

  const getChannelType = (channelId: number): NotificationChannelType => {
    const ch = channels.find((c) => c.id === channelId)
    return ch?.type ?? "email"
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="flex items-center gap-2">
            <Route className="size-5" />
            Routing Rules
          </CardTitle>
          <CardDescription>
            Define which alerts are sent to which channels based on severity.
          </CardDescription>
        </div>
        <Dialog open={createOpen} onOpenChange={(open) => setCreateOpen(open)}>
          <DialogTrigger
            render={
              <Button size="sm">
                <Plus className="mr-1 size-4" />
                Add Rule
              </Button>
            }
          />
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Routing Rule</DialogTitle>
              <DialogDescription>
                Route alerts of a specific severity to a notification channel.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <Label>Severity</Label>
                <Select
                  value={ruleSeverity}
                  onValueChange={(v) => {
                    if (v) setRuleSeverity(v)
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue>{ruleSeverity ? <span className="capitalize">{ruleSeverity}</span> : "Select severity"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {SEVERITIES.map((s) => (
                      <SelectItem key={s} value={s}>
                        <span className="capitalize">{s}</span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Channel</Label>
                {channels.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    Create a channel first before adding rules.
                  </p>
                ) : (
                  <Select
                    value={ruleChannel ? String(ruleChannel) : ""}
                    onValueChange={(v) => {
                      if (v) setRuleChannel(Number(v))
                    }}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue>{ruleChannel ? channels.find(c => c.id === ruleChannel)?.name ?? "Select channel" : "Select channel"}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {channels.map((ch) => (
                        <SelectItem key={ch.id} value={String(ch.id)}>
                          <span className="flex items-center gap-2">
                            {CHANNEL_TYPE_ICONS[ch.type]}
                            {ch.name}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>
            </div>
            <DialogFooter>
              <Button
                onClick={handleCreate}
                disabled={
                  !ruleChannel ||
                  !ruleSeverity ||
                  channels.length === 0 ||
                  createMutation.isPending
                }
              >
                {createMutation.isPending && (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                )}
                Create Rule
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="size-6 animate-spin text-muted-foreground" />
          </div>
        ) : rules.length === 0 ? (
          <EmptyState
            icon={<Route className="h-8 w-8" />}
            title="No routing rules configured."
            description="Add a rule to route alerts of specific severities to your notification channels."
          >
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1 size-4" />
              Add Rule
            </Button>
          </EmptyState>
        ) : (
          <div className="space-y-3">
            {rules.map((rule) => (
              <div
                key={rule.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="flex items-center gap-3">
                  <Badge
                    className={`capitalize ${SEVERITY_COLORS[rule.severity] ?? ""}`}
                  >
                    {rule.severity}
                  </Badge>
                  <span className="text-muted-foreground">&rarr;</span>
                  <div className="flex items-center gap-2">
                    <div className="flex size-6 items-center justify-center rounded bg-muted">
                      {CHANNEL_TYPE_ICONS[getChannelType(rule.channelId)]}
                    </div>
                    <span className="text-sm font-medium">
                      {getChannelName(rule.channelId)}
                    </span>
                  </div>
                </div>
                <ConfirmDialog
                  title="Delete Rule"
                  description={`Are you sure you want to remove the ${rule.severity} severity routing rule for ${getChannelName(rule.channelId)}?`}
                  actionLabel="Delete"
                  onConfirm={async () => { await deleteMutation.mutateAsync(rule.id) }}
                >
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={deleteMutation.isPending}
                    aria-label={`Delete routing rule for ${rule.severity} severity`}
                  >
                    <Trash2 className="size-4 text-destructive" />
                  </Button>
                </ConfirmDialog>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
