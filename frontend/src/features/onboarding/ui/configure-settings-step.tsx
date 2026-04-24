"use client"

import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldDescription,
  FieldError,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/ui"
import type { SettingsFormData } from "@/features/onboarding/hooks/use-onboarding"

interface ConfigureSettingsStepProps {
  formData: SettingsFormData
  errors: Record<string, string>
  isSaving: boolean
  onUpdateField: (field: keyof SettingsFormData, value: string) => void
  onSubmit: () => void
  onSkip: () => void
  onBack: () => void
}

const intervalOptions = [
  { value: "300", label: "Every 5 minutes" },
  { value: "900", label: "Every 15 minutes" },
  { value: "1800", label: "Every 30 minutes" },
  { value: "3600", label: "Every hour" },
  { value: "21600", label: "Every 6 hours" },
  { value: "86400", label: "Every 24 hours" },
]

export const ConfigureSettingsStep = ({
  formData,
  errors,
  isSaving,
  onUpdateField,
  onSubmit,
  onSkip,
  onBack,
}: ConfigureSettingsStepProps) => {
  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">Configure Monitoring</h2>
        <p className="text-sm text-muted-foreground">
          Set how Veilence-MX monitors package registries. You can change these later.
        </p>
      </div>

      <Field data-invalid={!!errors.discovery_scan_depth}>
        <FieldLabel>Top-N Packages to Monitor</FieldLabel>
        <Input
          type="number"
          min="1"
          max="10000"
          value={formData.discovery_scan_depth}
          onChange={(e) => onUpdateField("discovery_scan_depth", e.target.value)}
        />
        {errors.discovery_scan_depth && (
          <FieldError>{errors.discovery_scan_depth}</FieldError>
        )}
        <FieldDescription>
          Number of top packages by downloads to automatically monitor.
        </FieldDescription>
      </Field>

      <Field data-invalid={!!errors.monitoring_interval}>
        <FieldLabel>Monitoring Interval</FieldLabel>
        <Select
          value={formData.monitoring_interval}
          onValueChange={(val) => onUpdateField("monitoring_interval", val as string)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {intervalOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.monitoring_interval && (
          <FieldError>{errors.monitoring_interval}</FieldError>
        )}
        <FieldDescription>How often to check for new package releases.</FieldDescription>
      </Field>

      <Field data-invalid={!!errors.discovery_interval}>
        <FieldLabel>Discovery Interval</FieldLabel>
        <Select
          value={formData.discovery_interval}
          onValueChange={(val) => onUpdateField("discovery_interval", val as string)}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {intervalOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.discovery_interval && (
          <FieldError>{errors.discovery_interval}</FieldError>
        )}
        <FieldDescription>
          How often to discover new top-N packages from registries.
        </FieldDescription>
      </Field>

      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onBack} className="flex-1">
          Back
        </Button>
        <Button type="button" variant="ghost" onClick={onSkip}>
          Use Defaults
        </Button>
        <Button type="button" disabled={isSaving} onClick={onSubmit} className="flex-1">
          {isSaving ? "Saving..." : "Save & Continue"}
        </Button>
      </div>
    </div>
  )
}
