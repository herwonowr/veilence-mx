"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription, Button, Input, Field, FieldLabel, Badge, Switch, Separator, ConfirmDialog, EmptyState } from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/ui"
import {
  Plus,
  Trash2,
  Mail,
  Webhook,
  Loader2,
  BellRing,
  Route,
  Pencil,
  Zap,
} from "lucide-react"
import { useAuth, useCurrentWorkspaceRole, hasMinimumRole } from "@/core"
import {
  useChannels,
  useCreateChannel,
  useUpdateChannel,
  useDeleteChannel,
  useTestChannel,
  useRules,
  useCreateRule,
  useDeleteRule,
} from "@/features/notifications/hooks/use-channels"
import type { NotificationChannel, NotificationChannelType } from "@/domains/notifications"

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

export const ChannelsView = () => {
  const { currentWorkspace } = useAuth()
  const { role: currentRole } = useCurrentWorkspaceRole()
  const canManage = hasMinimumRole(currentRole, "admin")
  const workspaceId = currentWorkspace?.id ?? null

  const { data: channelsRes, isLoading: channelsLoading } = useChannels(workspaceId)
  const { data: rulesRes, isLoading: rulesLoading } = useRules(workspaceId)

  if (!canManage) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Notification Channels</h1>
          <p className="mt-1 text-muted-foreground">
            You do not have permission to manage notification channels. Admin access is required.
          </p>
        </div>
      </div>
    )
  }

  const channels = channelsRes?.data ?? []
  const rules = rulesRes?.data ?? []

  if (!workspaceId) {
    return (
      <div className="space-y-6">
        <h1 className="text-3xl font-bold">Notification Channels</h1>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Select a workspace to manage notification channels.
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
        workspaceId={workspaceId}
        channels={channels}
        loading={channelsLoading}
        canManage={canManage}
      />

      <Separator />

      {/* Rules Section */}
      <RulesSection
        workspaceId={workspaceId}
        rules={rules}
        channels={channels}
        loading={rulesLoading}
        canManage={canManage}
      />
    </div>
  )
}

// ─── Config Validation ─────────────────────────────────────────

const isConfigValid = (type: NotificationChannelType | "", config: string): boolean => {
  if (!type) return false
  try {
    const parsed = config ? JSON.parse(config) : {}
    switch (type) {
      case "email":
        return !!(parsed.host && parsed.port && parsed.from && parsed.to)
      case "slack":
        return !!parsed.webhookUrl
      case "webhook":
        return !!parsed.url
      default:
        return false
    }
  } catch {
    return false
  }
}

// ─── Channels Section ──────────────────────────────────────────

const ChannelsSection = ({
  workspaceId,
  channels,
  loading,
  canManage,
}: {
  workspaceId: string
  channels: NotificationChannel[]
  loading: boolean
  canManage: boolean
}) => {
  const [createOpen, setCreateOpen] = useState(false)
  const [channelName, setChannelName] = useState("")
  const [channelType, setChannelType] = useState<NotificationChannelType | "">("")
  const [channelConfig, setChannelConfig] = useState("")
  const [editChannel, setEditChannel] = useState<NotificationChannel | null>(null)
  const [editName, setEditName] = useState("")
  const [editConfig, setEditConfig] = useState("")

  const createMutation = useCreateChannel(workspaceId)
  const updateMutation = useUpdateChannel(workspaceId)
  const deleteMutation = useDeleteChannel(workspaceId)
  const testMutation = useTestChannel(workspaceId)

  const handleCreate = () => {
    if (!channelName || !channelType || !isConfigValid(channelType, channelConfig)) return
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

  const handleEditOpen = (channel: NotificationChannel) => {
    setEditChannel(channel)
    setEditName(channel.name)
    setEditConfig(channel.config)
  }

  const handleEditSave = () => {
    if (!editChannel || !editName || !isConfigValid(editChannel.type, editConfig)) return
    updateMutation.mutate(
      {
        id: editChannel.id,
        name: editName,
        config: editConfig,
        isActive: editChannel.isActive,
      },
      {
        onSuccess: () => {
          setEditChannel(null)
          setEditName("")
          setEditConfig("")
        },
      }
    )
  }

  return (
    <>
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
        {canManage && (
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
              <Field>
                <FieldLabel htmlFor="channel-create-name">Name</FieldLabel>
                <Input
                  id="channel-create-name"
                  placeholder="e.g., Team Slack"
                  value={channelName}
                  onChange={(e) => setChannelName(e.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel>Type</FieldLabel>
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
              </Field>
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
                disabled={!channelName || !channelType || !isConfigValid(channelType, channelConfig) || createMutation.isPending}
              >
                {createMutation.isPending && (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                )}
                Create Channel
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        )}
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
            {canManage && (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1 size-4" />
              Add Channel
            </Button>
            )}
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
                  {canManage && (
                  <>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => handleEditOpen(channel)}
                    aria-label={`Edit channel ${channel.name}`}
                    title="Edit channel"
                  >
                    <Pencil className="size-4" />
                  </Button>
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
                  </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>

    {/* Edit Channel Dialog */}
    <Dialog open={!!editChannel} onOpenChange={(open) => { if (!open) setEditChannel(null) }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Channel</DialogTitle>
          <DialogDescription>
            Update the channel name and configuration.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 pt-2">
          <Field>
            <FieldLabel htmlFor="channel-edit-name">Name</FieldLabel>
            <Input
              id="channel-edit-name"
              placeholder="e.g., Team Slack"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
            />
          </Field>
          {editChannel && (
            <ChannelConfigFields
              type={editChannel.type}
              config={editConfig}
              onChange={setEditConfig}
            />
          )}
        </div>
        <DialogFooter>
          <Button
            onClick={handleEditSave}
            disabled={!editName || !editChannel || !isConfigValid(editChannel?.type ?? "", editConfig) || updateMutation.isPending}
          >
            {updateMutation.isPending && (
              <Loader2 className="mr-2 size-4 animate-spin" />
            )}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    </>
  )
}

// ─── Channel Config Fields ──────────────────────────────────────

const ChannelConfigFields = ({
  type,
  config,
  onChange,
}: {
  type: NotificationChannelType
  config: string
  onChange: (config: string) => void
}) => {
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
        <div className="space-y-4">
          <Field>
            <FieldLabel htmlFor="channel-email-host">SMTP Host</FieldLabel>
            <Input
              id="channel-email-host"
              placeholder="smtp.example.com"
              value={parsed.host ?? ""}
              onChange={(e) => updateField("host", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="channel-email-port">SMTP Port</FieldLabel>
            <Input
              id="channel-email-port"
              placeholder="587"
              value={parsed.port ?? ""}
              onChange={(e) => updateField("port", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="channel-email-username">SMTP Username (optional)</FieldLabel>
            <Input
              id="channel-email-username"
              placeholder="user@example.com"
              value={parsed.username ?? ""}
              onChange={(e) => updateField("username", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="channel-email-password">SMTP Password (optional)</FieldLabel>
            <Input
              id="channel-email-password"
              type="password"
              placeholder="SMTP password or app password"
              value={parsed.password ?? ""}
              onChange={(e) => updateField("password", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="channel-email-from">From Address</FieldLabel>
            <Input
              id="channel-email-from"
              placeholder="alerts@example.com"
              value={parsed.from ?? ""}
              onChange={(e) => updateField("from", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="channel-email-to">To Address</FieldLabel>
            <Input
              id="channel-email-to"
              placeholder="team@example.com"
              value={parsed.to ?? ""}
              onChange={(e) => updateField("to", e.target.value)}
            />
          </Field>
        </div>
      )
    case "slack":
      return (
        <Field>
          <FieldLabel htmlFor="channel-slack-url">Slack Webhook URL</FieldLabel>
          <Input
            id="channel-slack-url"
            placeholder="https://hooks.slack.com/services/..."
            value={parsed.webhookUrl ?? ""}
            onChange={(e) => updateField("webhookUrl", e.target.value)}
          />
        </Field>
      )
    case "webhook":
      return (
        <div className="space-y-4">
          <Field>
            <FieldLabel htmlFor="channel-webhook-url">Webhook URL</FieldLabel>
            <Input
              id="channel-webhook-url"
              placeholder="https://api.example.com/webhook"
              value={parsed.url ?? ""}
              onChange={(e) => updateField("url", e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="channel-webhook-secret">Secret (optional)</FieldLabel>
            <Input
              id="channel-webhook-secret"
              type="password"
              placeholder="Signing secret for HMAC verification"
              value={parsed.secret ?? ""}
              onChange={(e) => updateField("secret", e.target.value)}
            />
          </Field>
        </div>
      )
    default:
      return null
  }
}

// ─── Rules Section ──────────────────────────────────────────────

const RulesSection = ({
  workspaceId,
  rules,
  channels,
  loading,
  canManage,
}: {
  workspaceId: string
  rules: Array<{ id: string; channelId: string; severity: string; isActive: boolean; createdAt: string }>
  channels: NotificationChannel[]
  loading: boolean
  canManage: boolean
}) => {
  const [createOpen, setCreateOpen] = useState(false)
  const [ruleChannel, setRuleChannel] = useState<string | null>(null)
  const [ruleSeverity, setRuleSeverity] = useState("")

  const createMutation = useCreateRule(workspaceId)
  const deleteMutation = useDeleteRule(workspaceId)

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

  const getChannelName = (channelId: string) => {
    const ch = channels.find((c) => c.id === channelId)
    return ch ? ch.name : `Channel #${channelId}`
  }

  const getChannelType = (channelId: string): NotificationChannelType => {
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
        {canManage && (
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
              <Field>
                <FieldLabel>Severity</FieldLabel>
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
              </Field>
              <Field>
                <FieldLabel>Channel</FieldLabel>
                {channels.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    Create a channel first before adding rules.
                  </p>
                ) : (
                  <Select
                    value={ruleChannel ? String(ruleChannel) : ""}
                    onValueChange={(v) => {
                      if (v) setRuleChannel(v)
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
              </Field>
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
        )}
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
            {canManage && (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1 size-4" />
              Add Rule
            </Button>
            )}
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
                {canManage && (
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
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
