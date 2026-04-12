"use client"

import { useState } from "react"
import type { APIKeyScope } from "@/domains/account"
import { Button } from "@/ui/components/button"
import { Input } from "@/ui/components/input"
import { Label } from "@/ui/components/label"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui/components/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/ui/components/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui/components/table"
import { Badge } from "@/ui/components/badge"
import { TableSkeleton, type SkeletonColumn } from "@/ui/feedback/table-skeleton"
import { TableError } from "@/ui/feedback/table-error"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { ConfirmDialog, type ConfirmDialogDetail } from "@/ui/feedback/confirm-dialog"
import { Key, Plus, Trash2, Copy, Check, Loader2 } from "lucide-react"
import { useApiKeys, useCreateApiKey, useDeleteApiKey } from "@/features/account/hooks/use-api-keys"

const SCOPE_OPTIONS: { value: APIKeyScope; label: string; description: string }[] = [
  { value: "read", label: "Read Only", description: "Can only read data (GET requests)" },
  { value: "write", label: "Read/Write", description: "Can read and write, but not delete" },
  { value: "admin", label: "Admin", description: "Full access to all operations" },
]

const scopeBadgeVariant = (scope: APIKeyScope): "secondary" | "default" | "destructive" => {
  switch (scope) {
    case "read":
      return "secondary"
    case "write":
      return "default"
    case "admin":
      return "destructive"
  }
}

export const ApiKeysView = () => {
  const { data: keysRes, isLoading, isError, refetch } = useApiKeys()
  const apiKeysSkeletonColumns: SkeletonColumn[] = [
    { width: "w-24", header: "Name" },
    { width: "w-20", header: "Key Prefix" },
    { width: "w-12", header: "Scope" },
    { width: "w-12", header: "Status" },
    { width: "w-20", header: "Last Used" },
    { width: "w-20", header: "Expires" },
    { width: "w-20", header: "Created" },
    { width: "w-8", header: "Actions" },
  ]

  const keys = keysRes?.data ?? []

  // Create form
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [keyName, setKeyName] = useState("")
  const [keyScope, setKeyScope] = useState<APIKeyScope>("read")
  const [expiresAt, setExpiresAt] = useState("")
  const [createError, setCreateError] = useState("")

  // Show key dialog
  const [showKeyDialogOpen, setShowKeyDialogOpen] = useState(false)
  const [newKeyValue, setNewKeyValue] = useState("")
  const [copied, setCopied] = useState(false)

  const createMutation = useCreateApiKey()
  const deleteMutation = useDeleteApiKey()

  // Delete confirmation state
  const [deleteTarget, setDeleteTarget] = useState<{
    id: number
    details: ConfirmDialogDetail[]
  } | null>(null)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateError("")
    try {
      const { data } = await createMutation.mutateAsync({
        name: keyName,
        scope: keyScope,
        expiresAt: expiresAt || undefined,
      })
      setNewKeyValue(data.apiKey)
      setCreateDialogOpen(false)
      setShowKeyDialogOpen(true)
      setKeyName("")
      setKeyScope("read")
      setExpiresAt("")
    } catch (err) {
      setCreateError(
        err instanceof Error ? err.message : "Failed to create API key"
      )
    }
  }

  const handleCopy = async () => {
    await navigator.clipboard.writeText(newKeyValue)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">API Keys</h1>
          <p className="text-sm text-muted-foreground">
            Manage API keys for programmatic access
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={(open) => setCreateDialogOpen(open)}>
          <DialogTrigger
            render={
              <Button>
                <Plus className="mr-2 size-4" />
                New API Key
              </Button>
            }
          />
          <DialogContent className="sm:max-w-md">
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Create API Key</DialogTitle>
                <DialogDescription>
                  Generate a new API key for programmatic access.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                {createError && (
                  <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive" role="alert">
                    {createError}
                  </div>
                )}
                <div className="space-y-2">
                  <Label htmlFor="key-name">Name</Label>
                  <Input
                    id="key-name"
                    placeholder="e.g. CI/CD Pipeline"
                    value={keyName}
                    onChange={(e) => setKeyName(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label>Scope</Label>
                  <div className="space-y-2">
                    {SCOPE_OPTIONS.map((option) => (
                      <label
                        key={option.value}
                        className={`flex cursor-pointer items-center gap-3 rounded-md border p-3 transition-colors ${
                          keyScope === option.value
                            ? "border-primary bg-primary/5"
                            : "border-border hover:bg-muted/50"
                        }`}
                      >
                        <input
                          type="radio"
                          name="key-scope"
                          value={option.value}
                          checked={keyScope === option.value}
                          onChange={() => setKeyScope(option.value)}
                          className="sr-only"
                        />
                        <div className="flex-1">
                          <p className="text-sm font-medium">{option.label}</p>
                          <p className="text-xs text-muted-foreground">
                            {option.description}
                          </p>
                        </div>
                        <Badge variant={scopeBadgeVariant(option.value)}>
                          {option.value}
                        </Badge>
                      </label>
                    ))}
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="key-expiry">Expiration (optional)</Label>
                  <Input
                    id="key-expiry"
                    type="datetime-local"
                    value={expiresAt}
                    onChange={(e) => setExpiresAt(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Leave empty for no expiration
                  </p>
                </div>
              </div>
              <DialogFooter>
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending && (
                    <Loader2 className="mr-2 size-4 animate-spin" />
                  )}
                  Create Key
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Show key dialog */}
      <Dialog open={showKeyDialogOpen} onOpenChange={(open) => setShowKeyDialogOpen(open)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>API Key Created</DialogTitle>
            <DialogDescription>
              Copy your API key now. You won&apos;t be able to see it again.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <div className="flex items-center gap-2 rounded-md bg-muted p-3">
              <code className="flex-1 break-all text-sm font-mono">
                {newKeyValue}
              </code>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={handleCopy}
                aria-label={copied ? "Copied to clipboard" : "Copy API key"}
              >
                {copied ? (
                  <Check className="size-4 text-green-600" />
                ) : (
                  <Copy className="size-4" />
                )}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setShowKeyDialogOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="size-5" />
            Your API Keys
          </CardTitle>
          <CardDescription>
            API keys are used to authenticate programmatic access to the API.
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <TableSkeleton columns={apiKeysSkeletonColumns} rows={5} />
          ) : isError ? (
            <TableError colSpan={8} onRetry={() => refetch()} />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key Prefix</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden lg:table-cell">Last Used</TableHead>
                  <TableHead className="hidden md:table-cell">Expires</TableHead>
                  <TableHead className="hidden md:table-cell">Created</TableHead>
                  <TableHead className="w-16">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {keys.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell className="font-mono text-sm">
                      {key.keyPrefix}...
                    </TableCell>
                    <TableCell>
                      <Badge variant={scopeBadgeVariant(key.scope ?? "admin")}>
                        {key.scope ?? "admin"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={key.isActive ? "secondary" : "outline"}
                      >
                        {key.isActive ? "Active" : "Revoked"}
                      </Badge>
                    </TableCell>
                    <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                      {key.lastUsedAt
                        ? new Date(key.lastUsedAt).toLocaleDateString()
                        : "Never"}
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
                      {key.expiresAt
                        ? new Date(key.expiresAt).toLocaleDateString()
                        : "Never"}
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
                      {new Date(key.createdAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() =>
                          setDeleteTarget({
                            id: key.id,
                            details: [
                              { label: "Name", value: key.name },
                              { label: "Key Prefix", value: `${key.keyPrefix}...` },
                              { label: "Scope", value: key.scope ?? "admin" },
                              { label: "Created", value: new Date(key.createdAt).toLocaleDateString() },
                            ],
                          })
                        }
                        aria-label={`Delete API key ${key.name}`}
                      >
                        <Trash2 className="size-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
                {keys.length === 0 && (
                  <TableEmptyState
                    colSpan={8}
                    icon={<Key className="h-8 w-8" />}
                    title="No API keys yet."
                    description="Create an API key to enable programmatic access to the Veilence-MX API."
                  >
                    <Button size="sm" onClick={() => setCreateDialogOpen(true)}>
                      <Plus className="mr-2 size-4" />
                      Create API Key
                    </Button>
                  </TableEmptyState>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Delete API Key Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Delete API Key?"
        description="Are you sure you want to delete this API key? Any applications using this key will lose access immediately. This action cannot be undone."
        details={deleteTarget?.details}
        actionLabel="Delete"
        onConfirm={async () => {
          if (deleteTarget) {
            await deleteMutation.mutateAsync(deleteTarget.id)
          }
        }}
      />
    </div>
  )
}

