"use client"

import { useState, useCallback } from "react"
import { workspaceSchema } from "@/domains/admin"
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

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      const result = workspaceSchema.safeParse(formData)
      if (!result.success) {
        const fieldErrors: Record<string, string> = {}
        for (const issue of result.error.issues) {
          const key = issue.path[0]
          if (typeof key === "string") {
            fieldErrors[key] = issue.message
          }
        }
        setErrors(fieldErrors)
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
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onUpdateField("name", e.target.value)}
        />
        {errors.name && <FieldError>{errors.name}</FieldError>}
      </Field>

      <Field>
        <FieldLabel>Slug</FieldLabel>
        <Input
          placeholder="acme-corp"
          value={formData.slug}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onUpdateField("slug", e.target.value)}
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
