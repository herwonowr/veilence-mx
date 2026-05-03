"use client"

import { useState } from "react"
import {
  SSOConfigList,
  SSOConfigForm,
  useCreatePlatformSSOConfig,
  useUpdatePlatformSSOConfig,
  AuthSettingsCard,
} from "@/features/sso-admin"
import type { SSOConfig, CreateSSOConfigRequest, UpdateSSOConfigRequest } from "@/domains/sso"

export const SSOAdminView = () => {
  const [editingConfig, setEditingConfig] = useState<SSOConfig | null>(null)
  const [showForm, setShowForm] = useState(false)

  const createMutation = useCreatePlatformSSOConfig()
  const updateMutation = useUpdatePlatformSSOConfig()

  const handleCreate = () => {
    setEditingConfig(null)
    setShowForm(true)
  }

  const handleEdit = (cfg: SSOConfig) => {
    setEditingConfig(cfg)
    setShowForm(true)
  }

  const handleCancel = () => {
    setEditingConfig(null)
    setShowForm(false)
  }

  const handleSubmit = (data: CreateSSOConfigRequest | UpdateSSOConfigRequest) => {
    if (editingConfig) {
      updateMutation.mutate(
        { id: editingConfig.id, ssoConfig: data as UpdateSSOConfigRequest },
        { onSuccess: () => setShowForm(false) }
      )
    } else {
      createMutation.mutate(data as CreateSSOConfigRequest, {
        onSuccess: () => setShowForm(false),
      })
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Security</h1>
        <p className="text-muted-foreground">
          Manage platform authentication and SSO providers.
        </p>
      </div>

      <AuthSettingsCard />

      {showForm ? (
        <SSOConfigForm
          existingConfig={editingConfig}
          isPending={createMutation.isPending || updateMutation.isPending}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
        />
      ) : (
        <SSOConfigList
          onCreateClick={handleCreate}
          onEditClick={handleEdit}
        />
      )}
    </div>
  )
}
