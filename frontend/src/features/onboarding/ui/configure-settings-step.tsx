"use client"

import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldDescription,
  FieldError,
} from "@/ui"
import type { SettingsFormData } from "@/features/onboarding/hooks/use-onboarding"

interface ConfigureSettingsStepProps {
  formData: SettingsFormData
  errors: Record<string, string>
  onUpdateField: (field: keyof SettingsFormData, value: string) => void
  onSubmit: () => void
  onSkip: () => void
  onBack: () => void
}

export const ConfigureSettingsStep = ({
  formData,
  errors,
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
          Set how Veilence-MX monitors package registries. You can change these later in Settings.
        </p>
      </div>

      <Field data-invalid={!!errors.monitoring_interval}>
        <FieldLabel htmlFor="onb-monitoring-interval">Monitoring Interval</FieldLabel>
        <Input
          id="onb-monitoring-interval"
          value={formData.monitoring_interval}
          onChange={(e) => onUpdateField("monitoring_interval", e.target.value)}
          placeholder="1h"
        />
        {errors.monitoring_interval && (
          <FieldError>{errors.monitoring_interval}</FieldError>
        )}
        <FieldDescription>Duration format (e.g., 30m, 1h, 6h)</FieldDescription>
      </Field>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field data-invalid={!!errors.discovery_scan_depth}>
          <FieldLabel htmlFor="onb-scan-depth">Discovery Scan Depth</FieldLabel>
          <Input
            id="onb-scan-depth"
            type="number"
            min={1}
            value={formData.discovery_scan_depth}
            onChange={(e) => onUpdateField("discovery_scan_depth", e.target.value)}
            placeholder="100"
          />
          {errors.discovery_scan_depth && (
            <FieldError>{errors.discovery_scan_depth}</FieldError>
          )}
          <FieldDescription>Top N packages per ecosystem per cycle</FieldDescription>
        </Field>

        <Field data-invalid={!!errors.discovery_interval}>
          <FieldLabel htmlFor="onb-discovery-interval">Discovery Interval</FieldLabel>
          <Input
            id="onb-discovery-interval"
            value={formData.discovery_interval}
            onChange={(e) => onUpdateField("discovery_interval", e.target.value)}
            placeholder="24h"
          />
          {errors.discovery_interval && (
            <FieldError>{errors.discovery_interval}</FieldError>
          )}
          <FieldDescription>Duration format (e.g., 12h, 24h, 7d)</FieldDescription>
        </Field>
      </div>

      <Button type="button" variant="outline" onClick={onSkip} className="w-full">
        Use Defaults
      </Button>

      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onBack} className="flex-1">
          Back
        </Button>
        <Button type="button" onClick={onSubmit} className="flex-1">
          Save & Continue
        </Button>
      </div>
    </div>
  )
}
