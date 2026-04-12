"use client"

import { Input } from "@/ui/components/input"
import { Label } from "@/ui/components/label"

interface FormFieldProps extends React.ComponentPropsWithRef<typeof Input> {
  label: string
  error?: string
  description?: string
}

export const FormField = ({
  label,
  error,
  description,
  id,
  ...inputProps
}: FormFieldProps) => (
  <div className="space-y-2">
    <Label htmlFor={id} className={error ? "text-destructive" : undefined}>
      {label}
    </Label>
    <Input
      id={id}
      aria-invalid={!!error}
      aria-describedby={
        error ? `${id}-error` : description ? `${id}-description` : undefined
      }
      className={error ? "border-destructive" : undefined}
      {...inputProps}
    />
    {error && (
      <p id={`${id}-error`} className="text-sm text-destructive" role="alert">
        {error}
      </p>
    )}
    {description && !error && (
      <p id={`${id}-description`} className="text-xs text-muted-foreground">
        {description}
      </p>
    )}
  </div>
)
