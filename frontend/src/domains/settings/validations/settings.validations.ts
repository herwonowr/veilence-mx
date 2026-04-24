import { z } from "zod"

const numericString = (label: string) =>
  z
    .string()
    .optional()
    .refine((val) => !val || /^\d+$/.test(val), {
      message: `${label} must be a number`,
    })

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const optionalCommaSeparatedEmails = z
  .string()
  .optional()
  .refine(
    (val) =>
      !val ||
      val
        .split(",")
        .map((e) => e.trim())
        .filter(Boolean)
        .every((e) => emailRegex.test(e)),
    { message: "All entries must be valid email addresses" }
  )

export const settingsSchema = z.object({
  monitoring_interval: numericString("Monitoring interval"),
  discovery_scan_depth: numericString("Discovery scan depth"),
  discovery_interval: numericString("Discovery interval"),
  discovery_auto_approve: z.enum(["true", "false"]).optional(),
  stale_auto_remove_months: numericString("Stale auto-remove months"),
  package_count_warning_threshold: numericString("Package count warning threshold"),
  email_digest_enabled: z.enum(["true", "false"]).optional(),
  email_digest_frequency: z.enum(["daily", "weekly"]).optional(),
  email_digest_recipients: optionalCommaSeparatedEmails,
})

export type SettingsFormData = z.infer<typeof settingsSchema>
