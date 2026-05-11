"use client"

import { useState, useCallback } from "react"
import { workspaceSchema } from "@/domains/admin"
import { parseFieldErrors } from "@/core"
import { Button, Input, Textarea, Field, FieldLabel, FieldError } from "@/ui"
import type { WorkspaceFormData } from "@/features/onboarding/hooks/use-onboarding"

interface CreateWorkspaceStepProps {
  formData: WorkspaceFormData
  onUpdateField: (field: keyof WorkspaceFormData, value: string) => void
  onSubmit: () => void
  onBack: () => void
}

export const CreateWorkspaceStep = ({
  formData,
  onUpdateField,
  onSubmit,
  onBack,
}: CreateWorkspaceStepProps) => {
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formSubmitted, setFormSubmitted] = useState(false)

  const validate = (fields: { name: string; slug: string; description: string }) => {
    if (!formSubmitted) return
    const result = workspaceSchema.safeParse(fields)
    setErrors(result.success ? {} : parseFieldErrors(result.error))
  }

  const handleSubmit = useCallback(
    (e: React.SyntheticEvent<HTMLFormElement>) => {
      e.preventDefault()
      setFormSubmitted(true)
      const result = workspaceSchema.safeParse(formData)
      if (!result.success) {
        setErrors(parseFieldErrors(result.error))
        return
      }
      setErrors({})
      onSubmit()
    },
    [formData, onSubmit]
  )

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">Create Your Workspace</h2>
        <p className="text-sm text-muted-foreground">
          A workspace is your team&apos;s container for monitoring packages.
        </p>
      </div>

      <Field>
        <FieldLabel>Workspace Name</FieldLabel>
        <Input
          placeholder="Acme Corp"
          value={formData.name}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => { onUpdateField("name", e.target.value); validate({ name: e.target.value, slug: formData.slug, description: formData.description }) }}
        />
        {errors.name && <FieldError>{errors.name}</FieldError>}
      </Field>

      <Field>
        <FieldLabel>Slug</FieldLabel>
        <Input
          placeholder="acme-corp"
          value={formData.slug}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => { onUpdateField("slug", e.target.value); validate({ name: formData.name, slug: e.target.value, description: formData.description }) }}
        />
        {errors.slug && <FieldError>{errors.slug}</FieldError>}
        <p className="text-xs text-muted-foreground">
          URL-safe identifier. Auto-generated from name.
        </p>
      </Field>

      <Field>
        <FieldLabel>
          Description <span className="text-muted-foreground">(optional)</span>
        </FieldLabel>
        <Textarea
          placeholder="Optional description"
          rows={2}
          value={formData.description}
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => onUpdateField("description", e.target.value)}
        />
      </Field>

      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onBack} className="flex-1">
          Back
        </Button>
        <Button
          type="submit"
          disabled={!formData.name || !formData.slug}
          className="flex-1"
        >
          Next
        </Button>
      </div>
    </form>
  )
}
